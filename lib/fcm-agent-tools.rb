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
  def install_file(location)
    return false unless diff(location, @filedata)

    STDERR.puts("#{$0}: Installing #{@filename} to #{location}")
    require 'tempfile'
    filedir = File.dirname(@filename)
    newfile = Tempfile.new(filedir)
    newfile.write(@filedata)
    File.rename(newfile.path, location)
    newfile.fsync
    newfile.close

    return true
  end
end
