# -*- encoding: utf-8 -*-
Gem::Specification.new do |s|
  s.name        = "rangeclient"
  s.version     = "0.0.11"
  s.platform    = Gem::Platform::RUBY
  s.authors     = ["Evan Miller"]
  s.email       = ["evan@squareup.com"]
  s.summary     = "Simple range client for ruby."
  s.description = "Use with range from https://github.com/square/libcrange"
  s.homepage    = "https://github.com/square/prodeng/tree/master/ruby_rangeclient"

  s.required_rubygems_version = ">= 1.3.6"

  s.default_executable = %q{er}
  s.executables = %W{ er range-compress range_split range_diff}

  s.files        = Dir.glob("lib/**/*") + Dir.glob("bin/*") + %w(README.md)
  s.extra_rdoc_files = ["LICENSE.md"]
  s.rdoc_options = ["--charset=UTF-8"]
end

