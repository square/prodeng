# -*- encoding: utf-8 -*-

Gem::Specification.new do |s|
  s.name        = "servicebuilder"
  s.version     = "0.0.3"
  s.platform    = Gem::Platform::RUBY
  s.authors     = ["Erik Bourget"]
  s.email       = ["github@squareup.com"]
  s.summary     = "Tool to build runit services from simple configuration files."
  s.description = "Tool to build runit services from simple configuration files."
  s.homepage    = "http://github.com/square/prodeng"

  s.required_rubygems_version = ">= 1.3.6"

  s.add_dependency "rdoc"
  s.default_executable = %q{servicebuilder}
  s.executables = %W{ servicebuilder }


  s.files        = Dir.glob("bin/*") + %w(README.md)
  s.extra_rdoc_files = ["LICENSE.md"]
  s.rdoc_options = ["--charset=UTF-8"]
end

