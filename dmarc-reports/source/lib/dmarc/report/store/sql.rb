require 'rubygems'
require 'sequel'
require 'nokogiri'
require 'pp'

module DMARC
  class Report
    class Store
      class SQL
      end
    end
  end
end

class DMARC::Report::Store::SQL
  attr :db

  def initialize(connstr)
    @db = Sequel.connect(connstr)
  end

  def bootstrap
    @db.create_table :ReportMetadata do
      primary_key :id
      String  :org_name
      String  :email
      String  :extra_contact_info
      String  :report_id
      Integer :begin_date # begin conflicts with kw in ruby
      Integer :end_date   # end conflicts with kw in ruby
      String  :error
    end

    @db.create_table:AlignmentType do
      primary_key :id
      String      :value
      index(:value, :unique => true)
    end
    @db[:AlignmentType].insert(:value => 'r')
    @db[:AlignmentType].insert(:value => 's')

    @db.create_table:DispositionType do
      primary_key :id
      String      :value
      index(:value, :unique => true)
    end
    @db[:DispositionType].insert(:value => "none")
    @db[:DispositionType].insert(:value => "quarantine")
    @db[:DispositionType].insert(:value => "reject")

    @db.create_table:DMARCResultType do
      primary_key :id
      String      :value
      index(:value, :unique => true)
    end
    @db[:DMARCResultType].insert(:value => "pass")
    @db[:DMARCResultType].insert(:value => "fail")

    @db.create_table:DKIMResultType do
      primary_key :id
      String      :value
      index(:value, :unique => true)
    end

    ["none","pass","fail","policy","neutral","temperror","permerror"].each do |e|
      @db[:DKIMResultType].insert(:value => e)
    end

    @db.create_table:SPFResultType do
      primary_key :id
      String      :value
      index(:value, :unique => true)
    end

    ["none","neutral","pass","fail","softfail","temperror","permerror"].each do |e|
      @db[:SPFResultType].insert(:value => e)
    end

    @db.create_table:DKIMAuthResultType do
      primary_key :id
      String      :value
    end

    @db.create_table:SPFAuthResultType do
      primary_key :id
      String      :value
    end

    @db.create_table:PolicyOverrideType do
      primary_key :id
      String      :value
    end

    @db.create_table :PolicyPublished do
      primary_key :id
      foreign_key(:report_metadata_id, :ReportMetadata, :key => :id)
      String      :domain
      foreign_key(:adkim, :AlignmentType, :key => :value,:type=>String)
      foreign_key(:aspf, :AlignmentType, :key => :value,:type=>String)
      foreign_key(:p, :DispositionType, :key => :value,:type=>String)
      foreign_key(:sp,:DispositionType, :key => :value,:type=>String)
      Integer     :pct
    end

    @db.create_table :PolicyOverrideReason do
      primary_key :id
      foreign_key(:report_metadata_id, :ReportMetadata, :key => :id)
      foreign_key(:type,:PolicyOverrideType,:key => :value)
      String      :comment
    end

    @db.create_table :SPFAuthResult do
      primary_key :id
      String      :domain
      foreign_key(:result, :SPFAuthResultType, :key => :value)
    end

    @db.create_table :DKIMAuthResult do
      primary_key :id
      String      :domain
      foreign_key(:result, :DKIMAuthResultType, :key => :value)
      String      :human_result
    end

    @db.create_table :Record do
      primary_key :id
      foreign_key(:report_metadata_id, :ReportMetadata, :key => :id)
      # row
      String  :source_ip
      Integer :count
      # policy evaluated
      foreign_key(:disposition, :DispositionType, :key => :value,:type => String)
      foreign_key(:dkim, :DMARCResultType, :key => :value,:type => String)
      foreign_key(:spf, :DMARCResultType, :key => :value,:type => String)
      # identifiers
      String      :envelope_to
      String      :header_from
      # auth_results XXX: to be implemented
      #foreign_key(:spf,  :SPFAuthResult, :key => :id)
      #foreign_key(:dkim, :DKIMAuthResult, :key => :id)
    end
  end

  def save_aggregate_report(o)
    m = o.ReportMetadata
    p = o.PolicyPublished
    m_to_save = {}
    @db[:ReportMetadata].columns.each { |c|
      next if(c.to_s == "id")
      m_to_save[c] = m.send(c)
    }
    report_metadata_id = @db[:ReportMetadata].insert(m_to_save)
    p_to_save = {}
    @db[:PolicyPublished].columns.each { |c|
      next if(c.to_s == "id")
      if(c.to_s == "report_metadata_id")
        p_to_save[c] = report_metadata_id
        next
      end
      p_to_save[c] = p.send(c)
    }
    @db[:PolicyPublished].insert(p_to_save)
    o.Records.each { |r|
      r_to_save = {}
      @db[:Record].columns.each { |c|
        next if (c.to_s == "id")
        if(c.to_s == "report_metadata_id")
          r_to_save[c] = report_metadata_id
          next
        end
        r_to_save[c] = r.send(c)
      }
      @db[:Record].insert(r_to_save)
    }
  end
end
