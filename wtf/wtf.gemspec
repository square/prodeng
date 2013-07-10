# -*- encoding: utf-8 -*-
Gem::Specification.new do |s|
  s.name        = "wtf"
  s.version     = "1.0.0"
  s.platform    = Gem::Platform::RUBY
  s.authors     = ["Erik Bourget"]
  s.email       = ["ewb@squareup.com"]
  s.summary     = "Tells you about whatever"
  s.description = ""
  s.homepage    = "http://github.com/square/prodeng"

  s.required_rubygems_version = ">= 1.3.6"

  s.add_dependency("json")
  s.add_dependency("rangeclient")
  s.add_dependency("sinatra")
  s.add_dependency("rangeclient")
  s.add_dependency("ascii_charts")
  s.executables = %W{ wtf wtf-web}

  s.files        = Dir.glob('lib/*') + Dir.glob('lib/providers/*') + Dir.glob('bin/*') + %w(README.md)
  #s.extra_rdoc_files = ["LICENSE.md"]
  s.rdoc_options = ["--charset=UTF-8"]
end

