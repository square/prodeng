become
==========

Become allows root to become other users without the side effects present in
sudo, su and other tools. Derived from djb's setuidgid.


Usage
-----

    root@box # become nobody id
    uid=99(nobody) gid=99(nobody) groups=99(nobody)
    root@box #

Useful when dropping privileges in an application runscript. For runit/dameontools:

    #!/bin/sh
    exec become nobody /path/to/myapp --foo bar --baz

Notes
-----

Sets supplementary groups as well.
