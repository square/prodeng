require 'rubygems'
require 'pp'
require 'resolv'
require 'rangeclient'

module WTF
  class RangeProvider < WTF::Provider
    NAME = "Range"

    def initialize(range_host = "range",
                   range_port = 80)
      @rc = Range::Client.new(:host => range_host, 
                              :port => range_port)
      @output = ''
    end

    def query(thing)
      clusters = @rc.expand('allclusters() & ' + thing)
      if clusters.length > 0
        newline "#{wtf_link thing} is a #{greenize 'cluster in Range'}."
        
        keys = @rc.expand('%' + thing + ':KEYS')
        memberlist 'It contains the following hosts', @rc.expand('%' + thing).sort

        keys.sort.each do |k|
          fact "It has #{k} set to", @rc.compress(@rc.expand('%' + thing + ':' + k))
        end
      end
      return @output
    end
  end
end
