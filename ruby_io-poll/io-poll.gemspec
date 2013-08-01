# -*- encoding: utf-8 -*-
Gem::Specification.new do |s|
  s.name        = "io-poll"
  s.version     = "0.0.4"
  s.platform    = Gem::Platform::RUBY
  s.authors     = ["Evan Miller"]
  s.email       = ["github@squareup.com"]
  s.summary     = "FFI bindings for poll(2) and select(2) emulation"
  s.description = "Ruby 1.8's IO.select() smashes the stack when given >1024 fds, and Ruby doesn't implement IO.poll()."
  s.homepage    = "http://github.com/square/prodeng"

  s.required_rubygems_version = ">= 1.3.6"

  s.add_dependency "ffi"

  s.files        = %w{lib/io/poll.rb} + %w(README.md)
  s.extra_rdoc_files = ["LICENSE.md"]
  s.rdoc_options = ["--charset=UTF-8"]
end

