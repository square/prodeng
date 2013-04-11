#!/usr/bin/ruby
require 'yaml'
require 'md5'

module FCM
  def self.bucketize(group_list)
    return Digest::MD5.hexdigest(group_list.to_s)
  end

  def self.write_files(nodes, directory, bucket)
    buckets_dir = File.join(directory, "buckets")
    bucket_dir = File.join(buckets_dir, bucket)
    needed_dirs = [directory, buckets_dir, bucket_dir]

    needed_dirs.each do |dir|
      unless File.directory?(dir)
        FileUtils.mkdir(dir)
      end
    end

    nodes[0].files.each do |fname, contents|
      File.open(File.join(bucket_dir, fname), 'w') do |f|
        contents.each { |line| f.puts(line.content) }
      end
    end

    link_nodes(nodes, directory, bucket)
  end

  def self.link_nodes(nodes, directory, bucket)
    buckets_dir = File.join(directory, 'buckets')
    bucket_dir = File.join(buckets_dir, bucket)
    nodes.each do |n|
      node_link = File.join(directory, n.name)
      if File.symlink?(node_link)
        if File.expand_path(File.readlink(node_link)) == File.expand_path(bucket_dir)
          next
        else
          File.symlink(bucket_dir, node_link + '.new')
          File.rename(node_link + '.new', node_link)
        end
      elsif File.exists?(node_link)
        raise "Non-symlink node file #{node_link}.  Will not delete"
      else
        File.symlink(bucket_dir, node_link)
      end
    end
  end

  class Config
    attr_accessor :datadir
  end

  class Node
    attr_reader :name, :groups, :files

    def initialize(name, config)
      @name = name
      @groups = []
      @files = {}
      @config = config
    end

    def add_group(group, path)
      return nil unless File.directory?(path)
      @groups.push(FCM::Group.new(group, path, @config))
    end

    def generate!
      @groups.each do |g|
        @files = g.apply(@files)
      end
      return @files
    end
  end

  class Group
    def initialize(name, directory, config)
      @name = name
      @transforms = {}
      @config = config

      Dir.open(directory) do |d|
        d.each do |f|
          next if f.start_with?(".")
          next unless File.file?(File.join(directory, f))
          @transforms[f] = FCM::Transform.new(f, File.join(directory, f), 
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

  class Transform
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
        FCM::Actions.apply(input, type, action_data, @group, @config)
      end
    end
  end

  class Line
    attr_accessor :group, :content
    def initialize(group, content)
      @group = group
      @content = content
    end
  end  

  module FCM::Actions
    def self.handle_dedup(input, action_data, group, config)
      unless action_data == nil
        raise "Parse error: DEDUP takes no arguments"
      end
      seen = {}
      output = []
      input.each do |line|
        output << line unless seen.has_key?(line.content)
        seen[line.content] = true
      end
      return output
    end

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
      input + [FCM::Line.new(group, action_data)]
    end

    def self.handle_include(input, action_data, group, config)
      unless action_data.is_a?(String)
        raise "Parse error: INCLUDE takes a filename"
      end

      lines = []
      data = File.readlines(File.join(config.datadir, "raw", action_data))
      data.each do |d|
        lines << FCM::Line.new(group, d)
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
          lines << FCM::Line.new(group, d)
        end
      end

      return input + lines
    end

    @handlers = %w[TRUNCATE APPEND INCLUDE DELETERE REPLACERE INCLUDELINE DEDUP].inject({}) do |h,type|
      h[type] = method("handle_#{type.downcase}")
      h
    end

    # input is an array
    def self.apply(input, type, action_data, group, config)
      raise "Invalid type" unless @handlers.has_key?(type)
      @handlers[type].call(input, action_data, group, config)
    end

  end
end

if __FILE__ == $0
  # invoke me from lib/
  require 'pp'

  node = FCM::Node.new("defaulthost")
  filemap = node.generate!
  filemap.each do |fname, data|
    data.each do |d|
      puts "FILE: #{fname}"
      puts d
    end
  end

  test_dir = File.join(File.dirname(__FILE__), "../testdata")

  g = FCM::Group.new("DEFAULT", File.join(test_dir, "groups/DEFAULT"))
  t = FCM::Transform.new(File.join(test_dir, "groups/DEFAULT/f.yaml"))
  input = ["Hello", "Goodbye"]
  puts t.apply(input)
end
