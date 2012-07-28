#!/usr/bin/ruby
require 'yaml'

class FcmConfig
  attr_accessor :datadir
end

class FcmNode
  attr_reader :name, :groups, :files

  def initialize(name, config)
    @name = name
    @groups = []
    @files = {}
    @config = config
  end

  def add_group(group, path)
    return nil unless File.directory?(path)
    @groups.push(FcmGroup.new(group, path, @config))
  end

  def generate!
    @groups.each do |g|
      @files = g.apply(@files)
    end
    return @files
  end
end

class FcmGroup
  def initialize(name, directory, config)
    @name = name
    @transforms = {}
    @config = config

    Dir.open(directory) do |d|
      d.each do |f|
        next if f.start_with?(".")
        next unless File.file?(File.join(directory, f))
        @transforms[f] = FcmTransform.new(f, File.join(directory, f), 
                                          name, @config)
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
  def initialize(name, file, group, config)
    @name = name
    @config = config
    @group = group
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
      FcmActions.apply(input, type, action_data, @group, @config)
    end
  end
end

class FcmLine
  attr_accessor :group, :content
  def initialize(group, content)
    @group = group
    @content = content
  end
end  

module FcmActions
  def self.handle_truncate(input, action_data, group, config)
    unless action_data == nil
      raise "Parse error: TRUNCATE takes no arguments"
    end
    []
  end

  def self.handle_append(input, action_data, group, config)
    unless action_data.is_a?(String)
      raise "Parse error: APPEND takes a string"
    end
    input + [FcmLine.new(group, action_data)]
  end

  def self.handle_include(input, action_data, group, config)
    unless action_data.is_a?(String)
      raise "Parse error: INCLUDE takes a filename"
    end

    lines = []
    data = File.readlines(File.join(config.datadir, "raw", action_data))
    data.each do |d|
      lines << FcmLine.new(group, d)
    end
  
    input + lines
  end
 
  def self.handle_replacere(input, action_data, group, config)
    unless action_data.has_key?('regex') and action_data.has_key?('sub')
      raise "Parse error: REPLACERE needs two named arguments"
    end

    regex = Regexp.new(action_data['regex'])

    input.each do |line|
      if regex.match(line.content)
        line.group = group
        line.content.gsub!(regex, action_data['sub'])
      end
    end
    return input
  end

  def self.handle_deletere(input, action_data, group, config)
    unless action_data.is_a?(String)
      raise "Parse error: DELETERE takes a string"
    end
    output = []
    regex = Regexp.new(action_data)
    input.each do |line|
      output += line unless regex.match(line.content)
    end
    
    return output
  end

  def self.handle_includeline(input, action_data, group, config)
    unless action_data.has_key?('regex') and action_data.has_key?('file')
      raise "Parse error: INCLUDELINE needs two named arguments"
    end
    
    lines = []
    File.open(File.join(config.datadir, "raw", action_data['file'])) do |f|
      data = f.readlines.grep(Regexp.new(action_data['regex']))
      data.each do |d|
        lines << FcmLine.new(group, d)
      end
    end

    return input + lines
  end

  @handlers = %w[TRUNCATE APPEND INCLUDE DELETERE REPLACERE INCLUDELINE].inject({}) do |h,type|
    h[type] = method("handle_#{type.downcase}")
    h
  end

  # input is an array
  def self.apply(input, type, action_data, group, config)
    raise "Invalid type" unless @handlers.has_key?(type)
    @handlers[type].call(input, action_data, group, config)
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
