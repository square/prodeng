#!/usr/bin/ruby

class FcmNode
  attr_accessor :name, :groups, :files
  def initialize(name)
    @name = name
    @groups = []
    @files = {}
  end
  
  def add_group(group, path)
    if File.directory?(path)
      @groups.push(FcmGroup.new(group, path))
    end
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
        next if f =~ /^\./
        next unless File.file?(File.join(directory, f))
        @transforms[f] = FcmTransform.new(File.join(directory, f))
      end
    end
  end

  # inputset is a map of { filename => [array of lines in file], etc }
  def apply(inputset)
    @transforms.each do |filename, t|
      unless inputset.has_key?(filename)
        inputset[filename] = []
      end
      inputset[filename] = t.apply(inputset[filename])
    end
  return inputset
  end

end

class FcmTransform
  require 'yaml'
  def initialize(file)
    @actions = []
    data = {}
    File.open(file) do |f|
      data = YAML.load(f.read)
    end
    unless data.is_a?(Array)
      raise "#{file} must be a yaml file with an array in it" 
    end
    
    data.each do |line|
      line.each do |type, rest|
        @actions.push(FcmAction.new(type, rest))
      end
    end
  end

  def apply(input)
    @actions.each do |a|
      input = a.apply(input)
    end
    return input
  end
end

class FcmAction
  def initialize(type, data)
    @type = type
    @data = data
  end

  # input is an array
  def apply(input)
    output = Array.new(input)
    case @type
    when "APPEND"
      output.push(@data)
    else
      raise "Invalid type"
    end
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

  g = FcmGroup.new("DEFAULT", "../testdata/groups/DEFAULT")
  t = FcmTransform.new("../testdata/groups/DEFAULT/f.yaml")
  input = ["Hello", "Goodbye"]
  puts t.apply(input)
end
