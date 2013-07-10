require 'rubygems'
require 'rangeclient'
require 'net/https'
require 'ascii_charts'
require 'pp'

module WTF
  class GraphiteProvider < WTF::Provider
    NAME = "Graphite"
    def initialize(range_host = "range",
                   range_port = "80")
      @rc = Range::Client.new(:host => range_host,
                              :port => range_port)
      @output = ''
    end
    
    def query(thing)
      gurl = @rc.expand('%{clusters(' + thing + 
                        ') & has(TYPE;core)}:GRAPHITE_URL')
      return '' unless gurl.length > 0
      gurl = gurl[0].gsub('"', '')
      newline "#{wtf_link thing} has data in #{greenize "Graphite"}."
      
      gthing = thing.gsub('.', '_')
      queries = {
        "CPU" => "scale(offset(nodes.#{gthing}.cpu.idle,-100.0),-1.0))",
        "Memory" => "offset(scale(divideSeries(sumSeries(nodes.#{gthing}.memory.memAvailReal,nodes.#{gthing}.memory.memBuffer,nodes.#{gthing}.memory.memCached),nodes.#{gthing}.memory.memTotalReal),-100),100)",
        "Load Average" => "nodes.#{gthing}.loadavg.1min"
      }

      queries.each do |k, v|
        get_graphite(gurl, k, v)
      end
      return @output
    end

    def get_graphite(graphiteurl,metricname, graphite_query)
      fulluri = URI::encode(graphiteurl + "/render?step=3600&target=" +
                            graphite_query +
                            "&from=-1d&until=now&rawData=true")
      uri = URI(fulluri)
      req = Net::HTTP::Get.new(fulluri)
      https = Net::HTTP.new(uri.host, uri.port)
      https.use_ssl = true
      https.verify_mode = OpenSSL::SSL::VERIFY_NONE
      response = https.request(req).body
      if response.include?('|')
        data = response.split('|')
        metadata = data[0].split(',')
        values = remove_none(data[1].chop.split(','))

        desired_datapoints = 24 # hours
        realdata = []
        i = 0
        values.each do |v|
          i += 1
          if i > (values.length / desired_datapoints)
            i = 0
            realdata << v
            next
          end
        end

        start = metadata[-3]
        finish = metadata[-2]
        step = metadata[-1]
        chartdata = realdata.map { |x| ['', x] }
        chart = AsciiCharts::Cartesian.new(chartdata)
        newline "Its #{metricname} over the last 24 hours:"
        chart.draw.each_line do |line|
          newline redize line.chomp
        end
      end
    end

    def remove_none(values)
      prev = 0.0
      return values.map { |value|
        if value == 'None' || value.nil?
          prev
        else
          prev = value.to_i
          value.to_i
        end
      }

    end

  end

end
