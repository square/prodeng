#Copyright (c) 2011, Yahoo! Inc. All rights reserved.

#This program is free software. You may copy or redistribute it under the same terms as Perl itself. Please see the LICENSE file included with this project for the terms of the Artistic License under which this project is licensed.


package Seco::MultipleCmd;
use strict;
use POSIX ":sys_wait_h";
use IO::Select;
use base qw(Seco::Class);

BEGIN {
    __PACKAGE__->_accessors(range => '',
                            hosts => [],
                            maxflight => 10,
                            maxflightpercent => 0,
                            next_ready => {},
                            nodes_in_flight => {},
                            global_timeout => 600,
                            timeout => 60,
                            unused_nodes => {},
                            select_timeout => 1,
                            write_buf => '',
                            rangehost => '',
                            not_ok => {},
                            times_run => {},
                            failed_nodes => {},
                            ok_nodes_cache => {},
                            read_select => undef,
                            error_select => undef,
                            write_select => undef,
                            maxread => 4096,
                            maxerror => 4096,
                            replace_hostname => '{}',
                            cmd => [],
                            yield_modify_cmd =>
                            sub {
                                my $self = shift;
                                my $node = shift; # Seco::Node
                                my @cmd = @{$self->{cmd}};
                                my $repl = $self->replace_hostname;
                                my $host = $node->hostname;
                                return map { s/$repl/$host/g; $_ } @cmd;
                            },
                            yield_node_start => sub { },
                            yield_node_finish => sub { },
                            yield_node_error => sub { },
                            nodes => {});
}

sub new {
    my $class = shift;
    my $self = $class->SUPER::new(@_);
    
    bless $self, $class;
    
    $self->{cmd} = [ $self->{cmd} ] unless ref $self->{cmd};
    
    $self->{maxflightpercent} = 0 if ($self->{maxflightpercent} < 0);
    # COMPAT: if we were passed a range, populate the hosts array
    if($self->{range}) {
        require Seco::Data::Range;
        $self->{r} = Seco::Data::Range->new;
        $self->{r}->{server} = $self->{rangehost}
          if($self->{rangehost});
        my @hosts = $self->{r}->expand($self->{range});
        $self->{hosts} = \@hosts;
    }
    
    exists($self->{hosts}) or die "No hosts passed";
    
    $self->evaluate_range;
    $self->{read_handles} = {};
    $self->{error_handles} = {};
    
    return $self;
}

sub evaluate_range {
    my $self = shift;
    
    foreach my $host (@{$self->{hosts}}) {
        my $node = Seco::MultipleCmd::Node->new(hostname => $host,
                                                cmd => $self->cmd,
                                                timeout => $self->timeout,
                                                maxread => $self->maxread,
                                                maxerror => $self->maxerror,
                                                write_buf => $self->write_buf);
        $self->nodes->{$node->hostname} = $node;
        unless($self->nodes_in_flight->{$node->hostname}) {
            $self->unused_nodes->{$node->hostname} = $node;
            $self->times_run->{$node->hostname} = 0;
        }
    }
    
    # Re-evaluate the percentage of nodes to keep in flight
    if ($self->maxflightpercent) {
        $self->maxflight( int ((scalar keys %{$self->nodes}) *
                               ($self->maxflightpercent / 100)) );
        $self->maxflight( 1 ) unless ($self->maxflight);
    }
}

sub run {
    my $self = shift;
    
    $self->read_select(IO::Select->new);
    $self->error_select(IO::Select->new);
    $self->write_select(IO::Select->new);
    my $start_time = time;
    my $last_eval = $start_time;

    while(
          ((!defined($self->global_timeout) or
            time < $start_time + $self->global_timeout)
           and
           (keys %{$self->unused_nodes} or
            keys %{$self->nodes_in_flight}))) {
        
        while(((scalar keys %{$self->nodes_in_flight}) <
               $self->maxflight) and
              keys %{$self->unused_nodes}) {
            unless($self->add_node) {
                sleep 1;
                last;
            }
        }
        
        # do I/O
        my @selected =
          $self->{write_select}->can_write($self->select_timeout);
        foreach my $handle (@selected) {
            my $node = $self->{write_handles}->{$handle};
            die "Can't find node for write handle $handle!" unless($node);
            $self->write_stdin($node);
        }
        
        @selected = $self->read_select->can_read($self->select_timeout);
        foreach my $handle (@selected) {
            my $node = $self->{read_handles}->{$handle};
            die "Can't find node for handle!" unless($node);
            $self->read_stdout($node);
        }
        
        @selected = $self->error_select->can_read($self->select_timeout);
        foreach my $handle (@selected) {
            my $node = $self->{error_handles}->{$handle};
            die "Can't find node for handle!" unless($node);
            $self->read_stderr($node);
        }

        # check for timed out nodes
        my $now = time;
        foreach my $node (values %{$self->nodes_in_flight}) {
            next if($self->{failed_nodes}->{$node->hostname}); # already seen
            if(defined($node->timeout) and
               ($node->started + $node->timeout) < $now) { # node timed out
                $self->error(($self->error||'') . "Timeout (" .
                             $node->timeout . " seconds elapsed for " .
                             $node->hostname . ")\n");
                $node->error("Timeout");
                $node->failed(1);
                $self->{failed_nodes}->{$node->hostname} = $node;
                $self->node_kill($node);
                $self->yield_node_error->($node);
            }
        }

        # reap some kids
        foreach my $pid (keys %{$self->{pids}}) {
            if(waitpid($pid, WNOHANG) > 0) {
                my $status = $? >> 8;
                my $node = $self->{pids}->{$pid};
                die "BUG / WEIRD: $pid does not have a node object!"
                  unless defined($node);

                # do final reads
                while($self->read_stdout($node)) {};
                while($self->read_stderr($node)) {};

                delete $self->{pids}->{$pid};
                delete $self->{nodes_in_flight}->{$node->hostname};
                $node->status($status);
                if($status) {
                    $node->error("Process exited $status");
                    $self->{failed_nodes}->{$node->hostname} = $node;
                    $self->yield_node_error->($node);
                }
                $self->yield_node_finish->($node);
                
            }
        }
    }
    
    # check global timeout
    if(scalar keys %{$self->nodes_in_flight} != 0 or
       keys %{$self->unused_nodes}) { # didn't finish, must be global timeout
        my $str = "Global timeout (" . $self->global_timeout .
          " seconds elapsed)\n";
        foreach my $node (values %{$self->nodes_in_flight},
                          values %{$self->unused_nodes}) {
            $node->reap if($self->{nodes_in_flight}->{$node->hostname});
            $self->{failed_nodes}->{$node->hostname} = $node;
        }
        $self->error(($self->error||'') . $str);
    }
    
    if(keys %{$self->failed_nodes}) {
        return $self->error(($self->error||'') . "(some unfinished)\n");
    } else { # success
        return 1;
    }
}

sub add_node {
    my $self = shift;
    
    return $self->error("No unused nodes left")
      unless(scalar keys %{$self->unused_nodes});
    
    my $nodename = undef;
    my $node = undef;
    
    my @nodes = sort { $self->times_run->{$a} <=> $self->times_run->{$b} }
      keys (%{$self->unused_nodes});
    my $time = time;
   
    my $i = 0;
    do {
        my $tmpnodename = $nodes[$i++];
        $self->next_ready->{$tmpnodename} = 0
          unless defined($self->next_ready->{$tmpnodename});
        if($time > $self->next_ready->{$tmpnodename}) {
            $nodename = $tmpnodename;
            $node = $self->{unused_nodes}->{$nodename};
            delete $self->{unused_nodes}->{$nodename};
            delete $self->{next_ready}->{$nodename};
        } else {
            return 0 if($i > $#nodes);
        }
    } while(!$node);

    my @cmd = $self->yield_modify_cmd->($self, $node);

    $node->cmd(\@cmd);

    my $pid = $node->start;
    $self->{pids}->{$pid} = $node;

    $self->read_select->add($node->read_handle);
    $self->error_select->add($node->error_handle);
    $self->write_select->add($node->write_handle)
      if(defined($self->{write_buf}));
    
    $self->{write_handles}->{$node->write_handle} = $node
      if(defined($self->{write_buf}));
    
    $self->{read_handles}->{$node->read_handle} = $node;
    $self->{error_handles}->{$node->error_handle} = $node;
    $self->{nodes_in_flight}->{$node->hostname} = $node;

    $self->yield_node_start->($node);
    $self->{times_run}->{$node->hostname}++;

    return 1;
}

sub node_kill {
    my $self = shift;
    my $node = shift;
    kill 9, $node->pid;
    $node->failed(1);
}

sub ok_nodes {
    my $self = shift;
    unless(%{$self->ok_nodes_cache}) {
        foreach my $node (keys %{$self->nodes}) {
            $self->ok_nodes_cache->{$node} = $self->nodes->{$node}
                unless $self->{failed_nodes}->{$node};
        }
    }
    return $self->{ok_nodes_cache};
}

sub ok {
    my $self = shift;
    return values %{$self->ok_nodes} if wantarray;
    return scalar values %{$self->ok_nodes};
}

sub failed {
    my $self = shift;
    return values %{$self->failed_nodes} if wantarray;
    return scalar values %{$self->failed_nodes};
}

sub failed_range {
    my $self = shift;
    my @hosts = map { $_->{hostname} } $self->failed;
    if($self->{range}) {
	require Seco::Data::Range;
	return Seco::Data::Range->new->compress(@hosts);
    } else {
	return join(',', @hosts);
    }
}

sub ok_range {
    my $self = shift;
    my @hosts = map { $_->{hostname} } $self->ok;
    if($self->{range}) {
	require Seco::Data::Range;
	return Seco::Data::Range->new->compress(@hosts);
    } else {
	return join(',', @hosts);
    }
}
	
sub write_stdin {
    my $self = shift;
    my $node = shift;
    
    my $handle = $node->write_handle;
    return undef unless $handle;
    my $select = IO::Select->new;
    $select->add($handle);
    return undef unless $select->can_write($self->select_timeout);
    
    my $offset = syswrite($handle, $node->write_buf, 4096,
                          $node->write_offset);
    
    if(!defined $offset || $offset < 0) {
        $node->error("Write error: $!");
        $self->{failed_nodes}->{$node->hostname} = $node;
        delete $self->{write_handles}->{$handle};
        $node->write_handle(undef);

        $self->write_select->remove($handle);
        close $handle;
        return undef;
    }

    $node->write_offset($node->write_offset + $offset);

    if($node->write_offset >= length $node->write_buf) {
        $self->write_select->remove($handle);
        close $handle;
        return undef;
    }
    return 1;
}

sub read_stdout {
    my $self = shift;
    my $node = shift;

    my $handle = $node->read_handle;
    return undef unless $handle;

    my $select = IO::Select->new;
    $select->add($handle);
    return undef unless $select->can_read($self->select_timeout);

    my $buf = '';
    my $bytes = sysread($handle, $buf, 4096);

    if(!defined $bytes || $bytes < 0) {
        $node->error("Read error: $!");
        $self->{failed_nodes}->{$node->hostname} = $node;
        $node->failed(1);

        delete $self->{read_handles}->{$handle};
        $node->read_handle(undef);
        $self->read_select->remove($handle);
        close $handle;

        return 0;
    } elsif($bytes == 0) { # success

        delete $self->{read_handles}->{$handle};
        $node->read_handle(undef);
        $self->read_select->remove($handle);
        close $handle;

        return 0;
    } else {
        return 0 if length($node->read_buf) >= $self->maxread;
        defined($node->read_buf) or $node->read_buf('');
        $node->{read_buf} .= $buf;
        return 1;
    }
}

sub read_stderr {
    my $self = shift;
    my $node = shift;

    my $handle = $node->error_handle;
    return undef unless $handle;

    my $select = IO::Select->new;
    $select->add($handle);
    return undef unless $select->can_read($self->select_timeout);

    my $buf = '';
    my $bytes = sysread($handle, $buf, 4096);

    if(!defined $bytes) {
        $node->error("Read error: $!");
        $self->{failed_nodes}->{$node->hostname} = $node;
        $node->failed(1);

        delete $self->{error_handles}->{$handle};
        $node->error_handle(undef);
        $self->error_select->remove($handle);
        close $handle;

        return 0;
    } elsif($bytes == 0) { # success
        delete $self->{error_handles}->{$handle};
        $node->error_handle(undef);
        $self->error_select->remove($handle);
        close $handle;

        return undef;
    } else {
        return 0 if length($node->error_buf) >= $self->maxerror;
        defined($node->error_buf) or $node->error_buf('');
        $node->{error_buf} .= $buf;
        return 1;
    }
}


package Seco::MultipleCmd::Node;
use strict;
use IPC::Open3;
use FileHandle;
use base qw(Seco::Class);

__PACKAGE__->_accessors(started => 0,
                        timeout => 60,
                        read_handle => undef,
                        error_handle => undef,
                        write_handle => undef,
                        maxread => 4096,
                        maxerror => 4096,
                        hostname => '',
                        cmd => [],
                        pid => undef, # don't set this
                        failed => 0,  # don't set this
                        read_buf => '',
                        error_buf => '',
                        write_buf => '',
                        write_offset => 0,
                        status => undef,
                       );

sub stringify_self {
    my $self = shift;
    my @cmd = @{$self->cmd};
    return $self->hostname . ": @cmd";
}

sub start {
    my $self = shift;
    unless(defined($SIG{PIPE})) {
        $SIG{PIPE} = sub {};
    }
    
    $self->hostname or return $self->error("No hostname given");
    
    my $pid;
    my ($writer, $reader, $error);
    
    $writer = FileHandle->new;
    $error = FileHandle->new;
    $reader = FileHandle->new;
    
    eval {
        $pid = open3($writer, $reader, $error, @{$self->cmd});
    };
    
    if($@) {
        return $self->error("Unable to run $self: $!\n");
    }
    
    $self->pid($pid);
    $self->write_handle($writer);
    $self->read_handle($reader);
    $self->error_handle($error);
    
    $self->started(time); # CLOCK IS TICKING
    
    return $pid;
}

sub reap {
    my $self = shift;
    close $self->{write_handle} if($self->{write_handle});
    close $self->{read_handle} if($self->{read_handle});
    close $self->{error_handle} if($self->{error_handle});
    kill 9, $self->pid;
    waitpid($self->pid, 0);
    return $? >> 8;
}

sub print_output {
    my $self = shift;
    my $hostname = $self->hostname;
    
    my $buf = $self->read_buf;
    if($buf) {
        $buf =~ s/^/$hostname: /mg;
        print STDOUT $buf;
    }
    
    $buf = $self->error_buf;
    if($buf) {
        $buf =~ s/^/$hostname(ERROR): /mg;
        print STDERR $buf;
    }
}
  
  1;

__END__

=pod

=head1 NAME

  Seco::MultipleCmd - module to perform batch operations

=head1 SYNOPSIS

  my $mcmd =
    Seco::MultipleCmd->new(range => "ks321000-20", # on these nodes,
                           cmd => "echo {}", # run this
                           maxflight => 10,        # forks to run at once
                           global_timeout => 600,  # in seconds
                           timeout => 60,          # timeout per node
                           write_buf => '',        # stdin for processes
                           replace_hostname => '{}', # replace this
                                                     # string with
                                                     # hostname
                          ) or die "Dead: $!";

  $mcmd->yield_node_start(sub { my $node = shift;
                               print "STARTING: $node\n"; } )
  $mcmd->yield_node_finish(sub { my $node = shift;
                                print $node->hostname . ": " .
                                      $node->read_buf;
                                print $node->hostname . "(STDERR): " .
                                      $node->error_buf; });
  $mcmd->yield_node_error(sub { my $node = shift;
                                print "$node ERROR: " . $node->error; } );
  $mcmd->yield_modify_cmd(sub { my $self = shift;
                                my $node = shift;
                                my @cmd = @{$self->cmd};
                                # alter @cmd to your heart's content
                                # to make it node-specific if you want
                                # see Seco::MultipleSsh for an example
                                return @cmd; }

  if(!$mcmd->run) { # timed out or otherwise failed
    die $mcmd->error;
  }

  while(my ($hostname, $nodeobj) = each (%{$mcmd->ok_nodes})) {
    my $read_buf = $nodeobj->read_buf;   # stdout
    my $error_buf = $nodeobj->error_buf; # stderr
  }

  while(my ($hostname, $nodeobj) = each (%{$mcmd->failed_nodes})) {
    my $read_buf = $nodeobj->read_buf;   # stdout
    my $error_buf = $nodeobj->error_buf; # stderr
  }

=head1 DESCRIPTION

Seco::MultipleCmd takes a (Seco::Range) range of nodes and a command.  It forks off processes out of a pool according to maxflight and runs your command on each node.  MultipleCmd will pass the given "write_buf" as stdin to each process.  Note that ssh collects passwords by reading out of /dev/tty and as such you will not be able to pass a password in this manner;  wrap MultipleCmd in expect if you require this functionality.




