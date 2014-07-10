mssh, mcmd
==========


Tools for running multiple commands and ssh jobs in parallel, and easily collecting the result

Usage
-----


<code>mssh -r  host01,host02,host03 "uname -r" -c</code>

```sh

Usage: mssh [options]
    -r, --range RANGE                Requires a configured Range::Client. Use --hostlist if you do not use range
        --hostlist x,y,z             List of hostnames to execute on
    -f, --file FILE                  List of hostnames in a FILE use (/dev/stdin) for reading from stdin
    -m, --maxflight 50               How many subprocesses? 50 by default
    -t, --timeout 60                 How many seconds may each individual process take? 0 for no timeout
    -g, --global_timeout 600         How many seconds for the whole shebang 0 for no timeout
    -c, --collapse                   Collapse similar output 
    -v, --verbose                    verbose 
    -d, --debug                      Debug output
    
```

Installing
-----

mssh is a rubygem: <code>gem install mssh</code>. http://rubygems.org/gems/mssh

BUGS/TODO
---------


 * Optionally Incorporate stderr into -c, with $?
 * allow commandline manipulation of ssh args
 * factor out redundancy between bin/mssh and bin/mcmd (cli module?)
 * incorporate range / foundation lookup syntax for -r
 * json output mode
 * to-file output mode
 * lots of rough spots, not super slick yet
 * needs testing real bad. 0.1 release

