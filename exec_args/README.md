exec\_args
==========

exec\_args is a simple binary which calls execvp on its args. It's essentially
a no-op, but can be useful as a wrapper to hold things like linux capabilities
or set*id bits or whatnot.



Usage
-----

    user@box $ exec_args ls * # no-op

    user@box $ cp exec_args grant_cap_net_bind; sudo setcap cap_net_bind_service=+ep grant_cap_net_bind

    user@box $ grant_cap_net_bind nc -l 80 # has ability to bind low ports without root


    #!/bin/sh
    exec become nobody grant_cap_net_bind /path/to/myapp --foo bar --baz

Notes
-----

You'll probably want to make a copy of this under another name when using it to
invoke permissions.

