servicebuilder
==========

servicebuilder - a tool for building runit directory structures from
a simple YAML configuration.  Tested with ruby 1.8.

Usage
-----
<code>servicebuilder -c CONF_DIR -s STAGING_DIRECTORY -d INSTALL_DIRECTORY</code>

Notes
-----

The STAGING_DIRECTORY is where your service directories will be created.  Runit
should *not* be monitoring this directory.  To have runit notice a service, it
will be symlinked into the INSTALL_DIRECTORY.  This directory should be the
directory monitored by runsvd.  Runit will then begin supervising the service.

CONF_DIR should be something like /etc/servicebuilder.d/ .  It must be
filled with config files that end in ".yaml".

Sample config file:

<code>
 mysql:
    run: mysqld -fg 
    sleep: 30
 sshd:
    run: sshd -fg
    log: svlogd -tt
    logsleep: 5
</code>

All "run" scripts and "log" scripts WILL BE PREPENDED WITH AN "exec"!
Your run script MUST RUN IN THE FOREGROUND or you'll create a fork bomb.
For more information about runit, see http://smarden.org/runit/ .

