#!/usr/bin/ruby
require 'yaml'

class FcmNode
  attr_reader :name, :groups, :files

  def initialize(name)
    @name = name
    @groups = []
    @files = {}
  end

  def add_group(group, path)
    return nil unless File.directory?(path)
    @groups.push(FcmGroup.new(group, path))
  end

  def generate!
    @groups.each do |g|
      @files = g.apply(@files)
    end
    return @files
  end
end

class FcmGroup
  def initialize(name, directory)
    @name = name
    @transforms = {}

    Dir.open(directory) do |d|
      d.each do |f|
        next if f.start_with?(".") || !File.file?(File.join(directory, f))
        @transforms[f] = FcmTransform.new(File.join(directory, f))
      end
    end
  end

  # inputset is a map of { filename => [array of lines in file], etc }
  def apply(inputset)
    @transforms.each do |filename, t|
      inputset[filename] = t.apply(inputset[filename] || [])
    end
    return inputset
  end

end

class FcmTransform
  def initialize(file)
    data = YAML.load_file(file)

    unless data.is_a?(Array)
      raise "#{file} must be a yaml file with an array in it"
    end

    @actions = data.inject([]) do |actions,line|
      actions + line.to_a
    end
  end

  def apply(input)
    @actions.inject(input) do |input, (type, action_data)|
      FcmActions.apply(input, type, action_data)
    end
  end
end

module FcmActions
  @datadir = File.join(File.dirname(__FILE__), "../testdata") # HACK

  def self.handle_truncate(input, action_data)
    unless action_data == nil
      raise "Parse error: TRUNCATE takes no arguments"
    end
    []
  end

  def self.handle_append(input, action_data)
    unless action_data.is_a?(String)
      raise "Parse error: APPEND takes a string"
    end
    input + [action_data]
  end

  def self.handle_include(input, action_data)
    unless action_data.is_a?(String)
      raise "Parse error: INCLUDE takes a filename"
    end

    input + File.readlines(File.join(@datadir, "raw", action_data))
  end

  def self.handle_replacere(input, action_data)
    unless action_data.has_key?('regex') and action_data.has_key?('sub')
      raise "Parse error: REPLACERE needs two named arguments"
    end

    regex = Regexp.new(action_data['regex'])

    input.map { |line| line.gsub(regex, action_data['sub']) }
  end

  def self.handle_deletere(input, action_data)
    unless action_data.is_a?(String)
      raise "Parse error: DELETERE takes a string"
    end
    output = []
    regex = Regexp.new(action_data)
    input.each do |line|
      output += line unless regex.match(line)
    end
    
    output
  end

  def self.handle_includeline(input, action_data)
    unless action_data.has_key?('regex') and action_data.has_key?('file')
      raise "Parse error: INCLUDELINE needs two named arguments"
    end
    
    File.open(File.join(@datadir, "raw", action_data['file'])) do |f|
      input + f.readlines.grep(Regexp.new(action_data['regex']))
    end
  end

  @handlers = %w[TRUNCATE APPEND INCLUDE DELETERE REPLACERE INCLUDELINE].inject({}) do |h,type|
    h[type] = method("handle_#{type.downcase}")
    h
  end

  # input is an array
  def self.apply(input, type, action_data)
    raise "Invalid type" unless @handlers.has_key?(type)
    @handlers[type].call(input, action_data)
  end

end

if __FILE__ == $0
  # invoke me from lib/
  require 'pp'

  node = FcmNode.new("defaulthost")
  filemap = node.generate!
  filemap.each do |fname, data|
    data.each do |d|
      puts "FILE: #{fname}"
      puts d
    end
  end

  test_dir = File.join(File.dirname(__FILE__), "../testdata")

  g = FcmGroup.new("DEFAULT", File.join(test_dir, "groups/DEFAULT"))
  t = FcmTransform.new(File.join(test_dir, "groups/DEFAULT/f.yaml"))
  input = ["Hello", "Goodbye"]
  puts t.apply(input)
end
