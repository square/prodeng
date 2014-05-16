package Seco::MD;
use base qw/Seco::Class/;
#Copyright (c) 2011, Yahoo! Inc. All rights reserved.

#This program is free software. You may copy or redistribute it under the same terms as Perl itself. Please see the LICENSE file included with this project for the terms of the Artistic License under which this project is licensed. 

use strict;
use warnings FATAL => qw/uninitialized/;

use Seco::Data::Range;
use Seco::MultipleCmd;
use Seco::Getopt;

use Net::Domain;
use File::Basename;
use List::Util            qw/shuffle/;
use File::Spec::Functions qw/splitpath canonpath file_name_is_absolute rel2abs file_name_is_absolute/;
use Carp                  qw/croak/;

use constant YES => 1;
use constant NO  => 0;

BEGIN {
    __PACKAGE__->_accessors(

        ssh =>
          [ '/usr/bin/ssh', '-o StrictHostKeyChecking=no', '-o ConnectTimeout=10', '-o BatchMode=yes', '-o ForwardAgent=yes', '-q', ],
        initssh =>
          [ '/usr/bin/ssh', '-o StrictHostKeyChecking=no', '-o ConnectTimeout=10', '-o BatchMode=yes', '-o ForwardAgent=yes', '-q', ],
        rsync => [
            '/usr/bin/rsync', '-a',  '--timeout','300','--progress',
        ],

        #### MD 
        range                   => undef,
        nodes                   => [],
        source                  => undef,
        destination_dir         => '/tmp',
        dedicated_seeds         => [],
        initial_seeds           => [],
        max_uploads             => 3,
        adjust_max_uploads      => YES,
        use_origin_host         => NO,
        max_retries_per_node    =>  0,
        max_failures            =>  5,
        verbose                 =>  0,
        debug                   =>  0,

        ### callbacks
        on_success              => sub { },
        on_failed               => sub { },

        #### mcmd
        mcmd_global_timeout     => undef,
        mcmd_node_timeout       => undef,
        mcmd_maxflight          => 200,

        #### rsync
        rsync_daemonmode        => undef,
        rsync_remote_path       => undef,
        rsync_ignore_existing   => '0',
        'rsync_password-file'   => '/home/rmas/.rsync/password',
        start_time              => time,

        #### MD state

        not_ok => [],
        ok     => [],
    );
}

sub _init {
    my $self = shift;
    
    croak "fatal: $self->{source} not readable"
        if (not -e $self->{source});

    croak "fatal: $self->{destination_dir} not an absolute path"
        unless ( file_name_is_absolute(canonpath $self->{destination_dir}) );

    $self->{source}             = canonpath $self->{source};
    $self->{destination_dir}    = canonpath( $self->{destination_dir} ) . "/";
    $self->{remote_src}        = ( splitpath $self->{source} )[2];

    # we have two shells to deal with
    $self->{remote_src}       = quotemeta quotemeta $self->{remote_src};


    # if we are using rsync running as a daemon
    if ( $self->{rsync_remote_path} ) {
        $self->{rsync_remote_path} .= '/';
        $self->{rsync_daemonmode} = YES;
        push( @{ $self->{rsync} },
            '--password-file', $self->{'rsync_password-file'} )
          if ( $self->{'rsync_password-file'} );
    }

    push ( @{$self->{rsync}}, '--dry-run' )
        if ( $self->{'dryrun'});

    $self->{r}          = Seco::Data::Range::->new;
    $self->{nodes}      = [$self->{r}->expand( $self->{range} )];

    $self->{nodestate}  =
      { map { $_ => { retries => '0', ok => '0', } } 
        (@{ $self->{nodes} }, @{$self->{initial_seeds}}, @{$self->{dedicated_seeds}}) }; 

    ### seeds/leechers 
    $self->{seeds}      = [];
    $self->{leechers}   = [@{$self->{nodes}}, @{$self->{initial_seeds}}, @{$self->{dedicated_seeds}}];

    $self->info( 'MD initialized: '
          . ' number of hosts: '
          . @{ $self->{leechers} }
          . "\n" );
    

    return $self;
}

##########
sub run {
    my $md       = shift;
    return 1 unless ( @{$md->{leechers}});


    $md->info('starting push');
    my $hostname = Net::Domain::hostfqdn;

    # some temporary variables
    my %dedicated_seeds = map { $_ => 1 } @{ $md->{dedicated_seeds} };

    # if no seeds, start the push to one node and use it as base 
    if ( not scalar @{ $md->{seeds} } ) {
        while (1) {
            my $i;

            ####  pick_initial_host
            $i = $hostname if ( $md->{use_self} );
            $i = ( shuffle( @{ $md->{initial_seeds} } ) )[0]
                    if ( ! defined $i && @{ $md->{initial_seeds} } );
            $i = ( shuffle( @{ $md->{dedicated_seeds} } ) )[0]
                    if ( ! defined $i && @{ $md->{dedicated_seeds} } && ! scalar @{$md->{initial_seeds}});
            $md->{leechers} = [ shuffle(@{$md->{leechers}}) ];
            $i = pop @{ $md->{leechers} } 
                    if ( ! defined $i && ! @{ $md->{dedicated_seeds} } 
                                      && ! @{$md->{initial_seeds}}   );
            #### initial_transfer
            my @cmd = ( @{$md->{rsync}},'-e',"@{$md->{initssh}}", $md->{source}, "$i:$md->{destination_dir}" );
            warn "@cmd";
            my $ret = system(@cmd);

            if ( $ret == 0 ) {
                push @{ $md->{seeds} }, $i;
                $md->info( "sweet: seeded to $i");
                $md->{nodestate}{$i}{ok}++;
                last;
            }
            croak "user interrupt" if (($ret >> 8 ) == 20);
            ### update nodestate
            $md->{nodestate}{$i}{retries}++;
            $md->_check_max_failures;
        }
    }
    push  @{$md->{rsync}},'-e', "'@{$md->{ssh}}'";
    # start duplication
    if ( @{ $md->{leechers} } ) {

        # map for distribution
        my %pull_map;
        $pull_map{$_} = ${$md->{seeds}}[0]
          foreach splice( @{ $md->{leechers} }, 0, $md->{max_uploads} );

        # construct the commands with the mapping
        my $mcmd = Seco::MultipleCmd::->new(
            range            => $md->{r}->compress( keys %pull_map ),
            maxflight        => $md->{mcmd_maxflight},
            cmd              => "",
            timeout          => $md->{mcmd_node_timeout},
            global_timeout   => $md->{mcmd_global_timeout},
            reevaluate_range => '0',
        );

        # note: $self is mcmd object
        $mcmd->yield_modify_cmd(
            sub {
                my ( $self, $node ) = @_;
                my $h = $node->hostname;

                # uh-oh
                croak "something went awry" unless $pull_map{$h};
                return $md->_rsync_cmd( $pull_map{$h}, $h );
            }
        );

        $mcmd->yield_node_start(
            sub {
                my $node = shift;
                $md->debug("execute:" . join( ' ', @{ $node->{cmd} } ) );
            }
        );

        $mcmd->yield_node_finish(
            sub {
                my $node = shift;
                my $now  = time;
                my $h    = $node->hostname;
                $md->{nodestate}{$h}{read_buf}  = $node->read_buf;
                $md->{nodestate}{$h}{error_buf} = $node->error_buf;


                if ( !$node->error ) {
                    $md->{nodestate}{$h}{ok}++;

                    # add it to seeds
                    push( @{ $md->{seeds} }, $h );

                    $md->info (  '(' .scalar @{$md->{seeds}} . '/' . scalar (keys %{ $md->{nodestate} }) . ') '. 
                      $h . ' transfer time: ' . ( $now - $node->{started} ) . ' total time: ' . ( $now - $md->{start_time} ) );

                    # release the seed we are leeching from and queue
                    # another leecher
                    return unless ( @{ $md->{leechers} } );
                    my $l = pop @{ $md->{leechers} };
                    $pull_map{$l} = $pull_map{$h};
                    $mcmd->{unused_nodes}{$l} = Seco::MultipleCmd::Node->new(
                        hostname => $l,
                        timeout  => $md->{mcmd_node_timeout},
                    );

                    # start leechers for this node and push the tasks to
                    # mcmd queue
                    $md->_adjust_max_uploads;
                    if ( not scalar @{$md->{dedicated_seeds}} || defined $dedicated_seeds{$node->hostname} ) { 
                        foreach my $l (
                            splice( @{ $md->{leechers} }, 0, $md->{max_uploads} )
                        )
                        {
                            $pull_map{$l} = $h;
                            $mcmd->{unused_nodes}{$l} =
                                Seco::MultipleCmd::Node->new(
                                    hostname => $l,
                                    timeout  => $md->{mcmd_node_timeout},
                             );
                        }
                    }
                    &{$md->on_success}($node->hostname);
                }
                else {
                    $md->{nodestate}{$h}{ok} = 0;

                    # check if we crossed our failure limits
                    $md->_check_max_failures;

                    # number of retries exceeded
                    if ( $md->{nodestate}{$h}{retries} >= $md->{max_retries_per_node} ) {
                        $md->debug( 'failed: ' 
                              . $h
                              . ' error: '
                              . $node->error . ":"
                              . $node->error_buf );
                        return;
                    }

                    # retry: push it back to the mcmd queue
                    $md->info( 'retry: ' . $h );
                    $mcmd->{unused_nodes}{$h} = Seco::MultipleCmd::Node->new(
                        hostname => $h,
                        timeout  => $md->{mcmd_node_timeout},
                    );
                    $md->{nodestate}{$h}{retries}++;
                    &{$md->on_failed}($node->hostname);
                }
                $mcmd->add_node;
            }
        );

        # run baby run
        $mcmd->run;
    }
    foreach my $node ( keys %{ $md->{nodestate} } ) {
        push @{ $md->{ok} }, $node
          if ( $md->{nodestate}{$node}{ok} );
        push @{ $md->{not_ok} }, $node
          if ( !$md->{nodestate}{$node}{ok} );
    }
}

sub _adjust_max_uploads {
    my $md = shift;
    my $current_max_uploads =
      int( ( @{ $md->{leechers} } ) / ( @{ $md->{seeds} } ) );
    if (   $md->{adjust_max_uploads}
        && $current_max_uploads < $md->{max_uploads} )
    {
        $md->{max_uploads} = $current_max_uploads;
        $md->{max_uploads} = 1 if ( $current_max_uploads < 1 );
        $md->debug( "  current leecher ratio:"
              . $current_max_uploads
              . 'adjusting max_uploads to'
              . $md->{max_uploads} );
    }
}

sub _check_max_failures {
    my $md = shift;
    my %error;
    foreach my $n ( keys %{ $md->{nodestate} } ) {
        $error{$n} = $md->{nodestate}{$n}
          if ( $md->{nodestate}{$n}{retries} >= $md->{max_retries_per_node}
            && !$md->{nodestate}{$n}{ok} );
    }
    if ( $md->{max_failures} >= 0 && keys %error >= $md->{max_failures} ) {
        info('number of failures exceeded limit of: ' . $md->{max_failures} );
        croak YAML::Syck::Dump( \%error );
    }
}

sub _rsync_cmd {
    my ( $md, $src, $dst, $src_path ) = @_;
    my @cmd;


    if ( not defined $src_path ) {
        $src_path =
            $md->{rsync_daemonmode}
          ? $md->{rsync_remote_path} . $md->{remote_src}
          : $md->{destination_dir} . $md->{remote_src};
    }

    @cmd = ( 
            @{ $md->{ssh} }, 
            $dst, '--',
            @{ $md->{rsync}},
            $src . ':' . $src_path,
            $md->{destination_dir},
           )
    unless $md->{rsync_daemonmode};

    @cmd = (
        @{ $md->{ssh} },
        $dst, '--',
        @{ $md->{rsync} },
        $src . '::' . $src_path,
        $md->{destination_dir},
    ) if $md->{rsync_daemonmode};
    
    return @cmd;
}

### Log4Syam
sub info  {  
    my $self = shift; 
    print  STDERR "info:  @_ \n" if ($self->{verbose} || $self->{debug}); 
}

sub debug {  
    my $self = shift; 
    print  STDERR "debug: @_ \n" if ($self->{debug});
}

1;

__END__

=head1 NAME

<Seco::MD> - <Centrally controlled File transfer>



=head1 VERSION

This documentation refers to <Seco::MD> version $version$

=head1 SYNOPSIS

    use Seco::MD;

    my $md = Seco::MD::->new (
			       range => $range,
			       source => $src,
			       destination_dir => $tmp,
                             );
    $md->run;

    foreach my $n ($md->ok()) {
	do_something;
    }

    foreach my $n ($md->not_ok()) {
	do_something;
    }


=head1 DESCRIPTION

  Seco::MD is used to transfer a file from a single destination 
  to "n" number of hosts using a peer-to-peer mechanism on
  top of rsync and ssh.

  The file transfer would roughly look like a nuclear fission.

  source ---> initial_seed <-- ( rsync over ssh ) n1 <-- n[4..6] 
                           |
			    <-- n2               
                           |
                            <-- n3

  The height/width of tree are controlled by maxflight and maxuploads
  per node.

  All the ssh commands are forked from central master.


=head1 SUBROUTINES/METHODS


=head1 DIAGNOSTICS

=head1 CONFIGURATION AND ENVIRONMENT


=head1 DEPENDENCIES

    Seco::Data::Range;
    Seco::MultipleCmd;
    Seco::Getopt;
    Net::Domain;
    File::Basename;
    List::Util;
    File::Spec;
    Carp     



=head1 INCOMPATIBILITIES




=head1 BUGS AND LIMITATIONS

documentation is incomplete. rtfs

=head1 AUTHOR

<Syam Puranam> ( <syam@yahoo-inc.com>) 

=head1 LICENCE AND COPYRIGHT

Copyright (c) 2011, Yahoo! Inc. All rights reserved
