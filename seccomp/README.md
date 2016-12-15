seccomp
==========

seccomp allows you to specify a seccomp kafel policy for a process
before running it.

Usage
-----

    seccomp -s "POLICY sample {}\nUSE sample DEFAULT ALLOW" true 
    seccomp -f kafel.policy true 

Notes
-----

[Kafel](https://github.com/google/kafel) is written by Google as a
method for specifying seccomp-bpf filters in an easy-to-understand
format.  See the [samples](https://github.com/google/kafel/tree/master/samples) for example policies.  Build with Bison 3.
