mssh, mcmd
==========


Tools for running multiple commands and ssh jobs in parallel, and easily collecting the result

Usage
-----


<code>mssh -r  host01,host02,host03 "uname -r" -c</code>

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

