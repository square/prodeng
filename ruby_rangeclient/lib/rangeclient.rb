#!/usr/bin/ruby

require 'rubygems'
require 'rest_client'
require 'cgi'

class Range::Client

  def initialize(options = {})
    default_host = 'range'
    default_host = ENV['RANGE_HOST'] if ENV.has_key?('RANGE_HOST')
    default_port = '80'
    default_port = ENV['RANGE_PORT'] if ENV.has_key?('RANGE_PORT')
    @options = {
      :host => default_host,
      :port => default_port,
    }.merge(options)
  end
  
  def expand(arg)
    escaped_arg = CGI.escape arg
    puts "http://#@options[:host]}:#{@options[:port]}/range/list?#{escaped_arg}"
    res = RestClient.get "http://#{@options[:host]}:#{@options[:port]}/range/list?#{escaped_arg}"
    return res.split "\n"
  end

  def compress(names)
    escaped_arg = CGI.escape names.join ","
    return RestClient.get "http://#{@options[:host]}:#{@options[:port]}/range/expand?#{escaped_arg}"
  end
end

if __FILE__ == $0
  require 'pp'
  rangehost = ARGV.shift
  rangearg = ARGV.shift
  r = Range::Client.new({:host => rangehost})
  hosts =  r.expand(rangearg)
  pp hosts
  pp r.compress(hosts)
end
