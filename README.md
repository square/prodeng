fcm - F configuration management
===

fcm is a configuration management system that aims to manage configuration, and
nothing more.

## Design goals

* Current state live on-hosts is never trusted.
* All data about what configuration a host should have is derived from group
  membership.
* fcm shall provide a trailer hitch on the front and on the back; no deep
  integration with other software is required and fcm will not interfere with
  other software.
* fcm shall not deal with dependency management in any way.  All operations are
  expected to happen at any time.

## How to run it

This command will read configs from ../testdata and dump all configs in 
/tmp/test:

    $ ./fcm-builder -d ../testdata/ -o /tmp/test

This command will read configs from ../testdata and show you what host1's passwd
file will look like:

    $ ./fcm-builder -d ../testdata/ -H host1 -f passwd
    root:x:0:0:root:/root:/bin/bash

## How it works

A YAML file specifies a group-to-host map.  A series of file on disks maps
groups to configuration file transformation operations.  When invoked,
*fcm-builder* reads the group-to-host map, builds a set of configuration files
based on group configuration, and produces a complete set of configuration files
for each host.

On-host, a lightweight agent - such as rsync in a loop - downloads host-specific
configurations and dumps them into a directory.  A set of lightweight
asynchronous agents are then free to read these files and do what they will with
them.  For example, an httpd.conf agent could atomically copy fcm's httpd.conf
into /etc/httpd and then reload apache if it was changed.

An agent for a file shall be a standalone executable program that will be passed
the name of its input file as its first argument.  These agents are expected to
do the right thing with regard to file atomicity, fsync, etc.

An example agent that installs a file into /tmp/hello:

    #!/usr/bin/ruby
    require 'tempfile'
    filename = ARGV[0]
    data = File.open(filename).read()
    file = Tempfile.new('/tmp/test')
    file.write(data)
    File.rename(file.path, "/tmp/hello")
    file.fsync
    file.close


## Trailer hitches?

Any good vehicle is designed to tow or be towed.  fcm does not assume that
you're using any particular CMDB, node classifier, or distribution mechanism.

The trailer hitch on the front is the group-to-host map YAML file; you're free
to edit it directly or to generate it from whatever source of truth you prefer.

The trailer hitch on the back is that fcm produces raw configuration files.  fcm
supplies a lightweight agent to run on end hosts, but you are similarly free to,
for example, copy them into a files/ directory in Puppet and use them from
there.

## File formats

Group transformations are applied in the order in which they appear in
hosts.yaml. 

DATADIR/hosts.yaml:

    - GROUPNAME:
        - host1
        - host2
        - host3
    - GROUP2:
        - host3
        - host4
        - hostZ

DATADIR/GROUPNAME/named.conf:

    - INCLUDE: "/var/fcm/files/named.conf.base"
    - APPEND: "controls {"
    - APPEND: "  inet 127.0.0.1 port 54 allow {any;}"
    - APPEND: "  keys { "rndc-key"; };"
    - APPEND: "};"

This will cause all hosts in GROUPNAME to start with a base named.conf
configuration, and then append some extra lines to it.  In this case, host1,2,3
will each get this named.conf.
