class FcmAgent
  require 'tempfile'
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

  def install_file(location)
    filedir = File.dirname(@filename)
    newfile = Tempfile.new(filedir)
    newfile.write(@filedata)
    File.rename(newfile.path, location)
    newfile.fsync
    newfile.close
  end
end
