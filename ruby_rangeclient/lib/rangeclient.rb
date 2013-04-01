#!/usr/bin/ruby

require 'rubygems'
require 'net/http'
require 'cgi'

class Range::Client
  attr_accessor :host, :port, :timeout

  # used to split hostnames into component parts for compression
  @@NodeRegx = /
                 ([-\w.]*?)                                # $1 - prefix
                 (\d+)                                     # $2 - start of range
                 (\.[-A-Za-z\d.]*[-A-Za-z]+[-A-Za-z\d.]*)? # optional domain
               /x;

  def initialize(options = {})
    @host = 'range'
    @host = ENV['RANGE_HOST'] if ENV.has_key?('RANGE_HOST')
    @host = @options[:host] if options.member?(:host)

    @port = '80'
    @port = ENV['RANGE_PORT'] if ENV.has_key?('RANGE_PORT')
    @port = @options[:port] if options.member?(:port)

    @timeout = 60
    @timeout = @options[:timeout] if options.member?(:timeout)
  end
  
  def expand(arg)
    escaped_arg = CGI.escape arg
    http = Net::HTTP.new(@host, @port)
    http.read_timout = @timeout
    req = Net::HTTP::Get.new('/range/list?' + escaped_arg)
    resp = http.request(req)
    return resp.body.split "\n"
  end


# Keep this extremely basic code for reference
#  def compress(nodes)
#    escaped_arg = CGI.escape nodes.join ","
#    return RestClient.get "http://#{@options[:host]}:#{@options[:port]}/range/expand?#{escaped_arg}"
#  end

# Take a page from the Perl Seco::Data::Range and perform this locally -- more efficient in both speed/size
# This was ported over from the Perl version, so it's not quite idiomatic ruby

  def compress(nodes)
    domain_tbl = {}
    no_domain_list = []
    nodes.each do |n|
      # If this is a quoted range, just compress it without collapsing
      return _simple_compress(nodes) if n =~ /^(?:"|q\()/

      # Break out host and key by domain, to enable {foo1,foo3}.bar.com grouping
      host, domain = n.split('.', 2)
      if domain
        domain_tbl[domain] ||= []
        domain_tbl[domain] << host
      else
        no_domain_list << host
      end
    end
    result = []
    # Range elements with no domain component do not group
    # just return
    if not no_domain_list.empty?
      result << _simple_compress(no_domain_list)
    end

    domain_tbl.keys.sort.each do |domain|
      r = _extra_compress(domain_tbl[domain])
      r.gsub!(/\.#{domain},/) {","}
      r.gsub!(/\.#{domain}$/) {""}
      if r=~ /,/
        r = "{#{r}}"
      end
      result << "#{r}.#{domain}" 
    end
    return result.join ","
  end

  def _extra_compress(nodes)
    domains = {}
    nodes = nodes.dup
    nodes.each do |node|
      node.gsub!(/^([a-z]+)(\d+)([a-z]\w+)\./) { "#{$1}#{$2}.UNDOXXX#{$3}." }
    end
    result = _simple_compress(nodes)
    result.each do |r|
      r.gsub!(/(\d+\.\.\d+)\.UNDOXXX/) {"{#{$1}}"}
      r.gsub!(/(\d+)\.UNDOXXX/) {"#{$1}"}
    end
    return result
  end

  def _simple_compress(nodes)
    # dedup nodes
    set = {}
    nodes.each do |node|
      set[node] = true
    end
    nodes = set.keys
    nodes = _sort_nodes(nodes)

    result = []
    prev_prefix, prev_digits, prev_suffix =  "", nil, ""
    prev_n = nil
    count = 0

    nodes.each do |n|
      if n =~ /\A#{@@NodeRegx}\z/
        # foo100abc => foo 100 abc
        prefix, digits, suffix = $1, $2, $3
        prefix = "" if prefix.nil?
        suffix = "" if suffix.nil?
      else
        prefix, digits, suffix = n, nil, nil
      end
      if (not digits.to_i.zero?) and
          (prefix == prev_prefix) and
          (suffix == prev_suffix) and
          (not prev_digits.nil?) and
          (digits.to_i == prev_digits.to_i + count + 1)
        count += 1
        next
      end
      
      if prev_n
        if count > 0
          result << _get_group(prev_prefix, prev_digits, count, prev_suffix)
        else
          result << prev_n
        end
      end
      prev_n = n
      prev_prefix = prefix
      prev_digits = digits
      prev_suffix = suffix
      count = 0
    end #nodes.each

    if count > 0
      result << _get_group(prev_prefix, prev_digits, count, prev_suffix)
    else
      result << prev_n
    end
    return result.join ","
  end

  def _sort_nodes(nodes)
    sorted = nodes.map { |n|
      # decorate-sort-undecorate
      # FIXME can this all be pushed into sort_by?
      n =~ /\A#{@@NodeRegx}\z/

      [ n,
        $1.nil? ? "" : $1,
        $2.nil? ? 0 : $2.to_i,
        $3.nil? ? "" : $3,
      ]
    }.sort_by { |e|
      [ e[1], e[3], e[2], e[0] ]
    }.map { |n|
      n[0]
    }
    return sorted
  end

  def _get_group(prefix, digits, count, suffix)
    prefix = "" if prefix.nil?
    group = sprintf("%s%0*d..%s",
                    prefix,
                    digits.to_s.length, 
                    digits.to_i,   # sometimes has leading zeroes
                    _ignore_common_prefix(digits, (digits.to_i + count).to_s)
                    )
    suffix = "" if suffix.nil?
    return group + suffix
  end

  def _ignore_common_prefix(start_pos, end_pos)
    len_start = start_pos.to_s.length
    return end_pos if len_start < end_pos.to_s.length
    pick = 0
    len_start.times do |i|
      pick = i
      # find the point at which the two strings deviate
      break if (start_pos[0..i] != end_pos[0..i])
    end
    # and return that substring prior to deviation
    return end_pos[pick..-1]
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
