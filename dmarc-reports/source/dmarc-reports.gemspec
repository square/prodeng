# -*- encoding: utf-8 -*-
Gem::Specification.new do |s|
  s.name        = "dmarc-reports"
  s.version     = "0.1.0"
  s.platform    = Gem::Platform::RUBY
  s.authors     = ["Syam Puranam"]
  s.email       = ["github@squareup.com"]
  s.summary     = "Interface to DMARC Reports"
  s.description = ""
  s.homepage    = "http://github.com/square/prodeng"

  s.required_rubygems_version = ">= 1.3.6"

  s.add_dependency("sequel", ">=3.40.0")
  s.add_dependency("sqlite3", ">=1.3.6")
  s.add_dependency("zipruby")
  s.add_dependency("mail")
  s.executables = %W{dmarc-reports-rest-api dmarc-aggregate-recieve}


  s.files        = Dir.glob('lib/**/*') + Dir.glob('bin/**/**/*') + %w(README.md)
  #s.extra_rdoc_files = ["LICENSE.md"]
  s.rdoc_options = ["--charset=UTF-8"]
end
