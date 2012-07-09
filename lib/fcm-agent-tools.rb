class FcmAgent
  def initialize()
    unless ARGV.length == 1
      raise "This program takes exactly one argument"
    end

    @filename = ARGV[0]

    unless File.exists?(@filename)
      raise "File #{@filename} does not exist"
    end
    
    @filedata = ""
    File.open(@filename) do |f|
      @filedata = f.read
    end
  end

  # true: there is a difference.  false: there is not
  def diff(file1, data)
    return true unless File.exists?(file1)
    f1 = File.open(file1)
    d1 = f1.read
    f1.close
    return ! (d1 == data)
  end

  # true: something was changed.  false: it was not
  # FIXME add permissions, install_cmd
  def install_file(location)
    return false unless diff(location, @filedata)

    STDERR.puts("#{$0}: Installing #{@filename} to #{location}")
    require 'tempfile'
    filedir = File.dirname(@filename)
    newfile = Tempfile.new(filedir)
    newfile.write(@filedata)
    # verify write. Only replace if on-disk file is what our buffer has
    if diff(newfile.path, @filedata)
      raise "Error: attempted to write file, but output data doesn't match"
      File.unlink(newfile.path)
    end
    File.rename(newfile.path, location)
    newfile.fsync
    newfile.close
    # If we're on linux, fsync the parent directory.
    arch, os = RUBY_PLATFORM.split('-')
    if os == "linux"
      d = File.new(filedir, "r")
      d.fsync
      d.close
    end
    return true
  end
end
