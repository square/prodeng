#Copyright (c) 2011, Yahoo! Inc. All rights reserved.

#This program is free software. You may copy or redistribute it under the same terms as Perl itself. Please see the LICENSE file included with this project for the terms of the Artistic License under which this project is licensed.

package Seco::Getopt;
use strict;
use Getopt::Long qw/:config require_order gnu_compat no_ignore_case/;
use Text::Wrap;

use base qw/Seco::Class/;

BEGIN {
    __PACKAGE__->_accessors(
        options  => {},
        default  => {},
        required => [],
        description => undef,
    );
    __PACKAGE__->_requires(qw/options/);
}

sub _init
{
    my $self = shift;
    $self->{options}->{'h|help'} = "Display this help";
    my %o;
    $self->usage unless GetOptions( \%o, keys %{ $self->{options} } );
    $self->usage if ( $o{h} );

    foreach my $key ( keys %{ $self->{default} } ) {
        exists $o{$key} or $o{$key} = $self->{default}->{$key};
    }

    my $bad = '';
    for ( @{ $self->{required} } ) {
        $bad .= "Option $_ is required\n" unless ( defined $o{$_} );
    }
    if ($bad ne '') {
        $bad =~ s/\s+$//m;
        $self->usage($bad);
    }

    $self->{o} = \%o;
    $self;
}

sub get
{
    my $self = shift;
    my $opt  = shift;
    die "No argument supplied" unless ( defined $opt );
    return $self->{o}->{$opt};
}

sub set
{
    my $self = shift;
    my $opt  = shift;
    my $val  = shift;
    die "No argument supplied" unless ( defined $opt and defined $val );
    $self->{o}->{$opt} = $val;
}

sub usage
{
    my $self = shift;

    my $msg = shift;
    $msg = "\n$msg\n" if ($msg);
    $msg ||= '';

    print "Usage: $0 [options]\n";
    if ($self->{description}) {
        $Text::Wrap::columns = 80;
        print wrap('', "             ", "Description: ",
                   $self->{description}, "\n");
    }
    print "Options:\n";
    my @array;
    foreach my $key ( keys %{ $self->{options} } ) {
        my $default = '';
        my ( $left, $right ) = split /[=:]/, $key;
        if ( exists $self->{default}->{$left} ) {
            my $val = $self->{default}->{$left};
            if ( ref $val and ref $val eq "ARRAY" ) {
                $default = "[" . ( join ',', @$val ) . "]";
            } 
            if ( ref $val and ref $val eq "HASH" ) {
                $default = "[" . join (',', map { "$_:$val->{$_}" } keys %$val ) . "]";
            } else {
                if ($key !~ /=/) {
                    $default = ($val) ? "[yes]" : "[no]";
                } else {
                    $default = "[$val]";
                }
            }
        } elsif ( $left =~ /(.*)\|(.*)/ and defined($1) ) {
            my $val = $self->{default}->{$1};
            if ( defined $val ) {
                if ( ref $val and ref $val eq "ARRAY" ) {
                    $default = "[" . ( join ',', @$val ) . "]";
                } 
                if ( ref $val and ref $val eq "HASH" ) {
                    $default = "[" . join (',', map { "$_:$val->{$_}" } keys %$val ) . "]";
                 } else {
                    if ($key !~ /=/) {
                        $default = ($val) ? "[yes]" : "[no]";
                    } else {
                        $default = "[$val]";
                    }
                }
            }
        }

        $default .= " (required)" if (grep /^$key$/, @{$self->{required}});
        my ( $a, $b ) = split /\|/, $left;
        if ($b) {
            $b = substr("{no}$b",0,-1) if ($b =~ /\!$/);  # negatable option
            $left = "-$a, --$b";
        } elsif ( $a =~ /^.$/ )
        {    # if $a is only a single char, use only one '-'
            $left = "-$a      ";
        } else {
            $a = substr("{no}$a",0,-1) if ($a =~ /\!$/);  # negatable option
            $left = "    --$a";
        }

        $left = substr( $left . ( ' ' x 25 ), 0, 25 );
        push @array, "$left " . $self->{options}->{$key} . " $default\n";
    }
    print sort @array;

    die "$msg\n";

}

1;

=pod

=head1 NAME

Seco::Getopt - A friendly wrapper around Getopt::Long

=head1 SYNOPSIS

  use Seco::Getopt;

  my $opt = Seco::Getopt->new(
    options => {
	  'r|range=s'    => 'Seco range of nodes',
	  'l|limit=i'    => 'Limit results to this number',
	  'y|yes'        => 'Yes, really do something',
    },
    default => {
	  l => 1000,
    },
    required => [ 'r' ],
    description => 'Short description of the program, which is optional';
  );

  my $range = $opt->get('r');
  my $limit = $opt->get('l');
  die "not doing anything" unless $opt->get('y');

=head1 DESCRIPTION

Friendly wrapper around Getopt::Long that exposes an easy to use interface.
Automatically generates usage and adds a --help

Options passed to Getopt::Long are:

  require_order
  gnu_compat
  no_ignore_case

See SYNOPSIS for usage.

=head1 AUTHOR

Erik Bourget, E<lt>F<ebourget@yahoo-inc.com>E<gt>

=head1 SEE ALSO

Getopt::Long(3)

=cut
