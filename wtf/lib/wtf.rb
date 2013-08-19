#!/usr/bin/ruby

require 'rubygems'
require 'provider'
require 'providers/ascii'
require 'providers/range'
require 'providers/dns'

module WTF
  class Wtf
    attr_accessor :output
    def initialize
      @output = {}
      @providers = [
                    WTF::ASCIIProvider.new,
                    WTF::RangeProvider.new,
                    WTF::DNSProvider.new]
    end

    def newline(provider, text)
      @output[provider] = '' unless @output.has_key?(provider)
      @output[provider] += text + "\n"
    end

    def query(thing)
      threads = []
      @providers.each do |provider|
        threads << Thread.new do
          data = provider.query(thing)
          if data
            data.each_line do |line|
              newline(provider, line.chomp)
            end
          end
        end
      end
      threads.each do |t|
        t.join
      end
    end
  end
end

