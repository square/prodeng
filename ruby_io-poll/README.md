io-poll
==========

FFI bindings for poll(2) and select(2) emulation
Ruby 1.8's IO.select() smashes the stack when given >1024 fds, and Ruby doesn't implement IO.poll().


Usage
-----

require 'io/poll'
read_fds  = [STDIN]
write_fds = [STDOUT]
err_fds   = []
poll_period = 60
read_fds, write_fds, err_fds = IO.select_using_poll(read_fds, write_fds, err_fds, poll_period)


BUGS/TODO
---------

 * This is a hack.

