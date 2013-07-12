require 'rubygems'
require 'sequel'
require 'nokogiri'

module DMARC
  class Report
    class Aggregate
    end
  end
end

class DMARC::Report::Aggregate
  attr_accessor :ReportMetadata,:PolicyPublished,:Records
  class ReportMetadata
    attr_accessor :org_name,:email,:extra_contact_info,:report_id,:error,:begin_date,:end_date
  end
  class PolicyPublished
    attr_accessor :domain,:adkim,:aspf,:p,:sp,:pct
  end
  class Record
    attr_accessor :source_ip,:count,:header_from,:envelope_to,:disposition,:dkim,:spf
  end

  def initialize
    @ReportMetadata = ReportMetadata.new
    @PolicyPublished = PolicyPublished.new
    @Records = []
  end

  def parse(str)
    f = Nokogiri::XML(str)
    # parse metadata
    ["begin","end"].each { |s|
      # calls accessor assignment method
      @ReportMetadata.send(
        "#{s}_date=",
        f.root.at_xpath("/feedback/report_metadata/date_range/#{s}").text.to_i)
    }

    ["org_name","email","extra_contact_info","report_id","error"].each { |s|
      if(e = f.root.at_xpath("/feedback/report_metadata/#{s}"))
        @ReportMetadata.send("#{s}=",e.text)
      end
    }

    # parse published policy
    @PolicyPublished = PolicyPublished.new
    ["domain","adkim","aspf","p","sp"].each { |s|
      if(e = f.root.at_xpath("/feedback/policy_published/#{s}"))
        @PolicyPublished.send("#{s}=",e.text)
      end
    }
    @PolicyPublished.pct = f.root.at_xpath("/feedback/policy_published/pct").text.to_i

    # parse records
    @Records = []
    f.root.xpath("//feedback//record").each { |e|
      r = Record.new
      r.source_ip   = e.at_xpath("./row/source_ip").text
      r.count       = e.at_xpath("./row/count").text.to_i
      r.header_from = e.at_xpath("./identifiers/header_from").text

      if(e.at_xpath("./identifiers/envelope_to"))
        r.envelope_to = e.at_xpath("./identifier/envelope_to").text
      end

      ["disposition","dkim","spf"].each do |s|
        r.send("#{s}=",e.at_xpath("./row/policy_evaluated/#{s}").text)
      end

      @Records.push(r)
    }

  end
end
