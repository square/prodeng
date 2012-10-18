nolimit
==========

Set hard and soft rlimits to either infinity, or as high as possible
Derived in spirit from djb's daemontools, tries to follow some similar conventions.


Usage
-----

    root@box # nolimit myprog --myprogargs

Useful when setting limits in an application runscript. For runit/dameontools:

    #!/bin/sh
    exec nolimit /path/to/myapp --foo bar --baz

Notes
-----

Must run as root. Invoke before dropping privileges.
