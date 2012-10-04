# -*- encoding: utf-8 -*-
Gem::Specification.new do |s|
  s.name        = "elvis"
  s.version     = "0.0.1"
  s.platform    = Gem::Platform::RUBY
  s.authors     = ["Evan Miller"]
  s.email       = ["evan@squareup.com"]
  s.summary     = "Elvis impersonates users"
  s.description = "When you're root, sometimes it's handy to drop privileges temporarily to impersonate a user"
  s.homepage    = "https://github.com/square/prodeng"

  s.required_rubygems_version = ">= 1.3.6"

  s.files        = Dir.glob("lib/**/*") + %w(README.md)
  s.extra_rdoc_files = ["LICENSE.md"]
  s.rdoc_options = ["--charset=UTF-8"]
end

