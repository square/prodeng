require 'rubygems'
require 'resolv'
require 'net/http'
require 'json'

module WTF
  class DNSProvider < WTF::Provider
    NAME = "DNS"

    def initialize()
      @output = ''
      @resource = RestClient::Resource.new 'http://whois.arin.net/rest'
    end

    def get_resource(rootel, query)
      begin
        @resource["#{rootel}/#{query}.json"].get
      rescue RestClient::ResourceNotFound
        raise "uhoh"
      end
    end

    def arin_query(ip)
      begin
        uri = URI("http://whois.arin.net/rest/ip/#{ip}.json")
        http = Net::HTTP.new(uri.host, uri.port)
        req = Net::HTTP::Get.new(uri.path)
        resp = http.start { |cx| cx.request(req) }
        if resp.code == "200"
          return JSON.parse(resp.body)
        end
      end
      return nil
    end

    def query(thing)
      if thing == "wtf"
        return ''
      end
      begin
        addr = Resolv.getaddress(thing)
        newline "#{wtf_link thing} is in #{greenize 'DNS'}."
        fact "Its address is", addr
        data = arin_query(addr)
        if data
          fact "It is owned by", data['net']['name']['$']
          netblock = data['net']['netBlocks']['netBlock']
          nb = netblock['startAddress']['$'] + "/" + netblock['cidrLength']['$']
          fact "Its netblock is ", nb
          if data['net']['registrationDate']
            fact "It was assigned on", data['net']['registrationDate']['$']
          end
        end
      rescue Resolv::ResolvError => e
      end

      return @output
    end
  end
end
