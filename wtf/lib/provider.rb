require 'rubygems'
require 'term/ansicolor'
include Term::ANSIColor

module WTF
  class Provider
    attr_accessor :name
    def newline(text)
      @output = "" if @output.nil?
      @output += text + "\n"
    end

    def wtf_link(text)
      reset + bold + yellow + text + reset
    end

    def url(label, text)
      newline label + ": " + reset + bold + cyan + text + reset
    end

    def fact(label, text)
      newline label + ": " + reset + bold + green + text + reset
    end

    def memberlist(label, members)
      newline label + ":"
      members.each do |m|
        newline "   " + reset + bold + yellow + m + reset
      end
    end

    def greenize(text)
      reset + bold + green + text + reset
    end

    def redize(text)
      reset + bold + magenta + text + reset
    end

    def blueize(text)
      reset + bold + cyan + text + reset
    end
  end
end
