require 'rubygems'
require 'sequel'
require 'pp'


module DMARC
  class Report
    class View
      class SQL
      end
    end
  end
end

class DMARC::Report::View::SQL
  attr :db

  def initialize()
    @db = Sequel.connect('sqlite://dmarc-reports.db')
  end

  def get_all_reports(limit)
    if(limit)
      return @db[:ReportMetadata].limit(limit).all
    else
      return @db[:ReportMetadata].all
    end
  end

  def get_all_reports_by_date(s,e)
    return @db[:ReportMetadata].where{:begin >= s && :end <= e}
  end

  def get_all_reports_filtered(k,v)
    return @db[:ReportMetadata].filter(k.to_sym => v).all
  end

  def get_all_reports_paginated(idx,count)
    ds = @db[:ReportMetadata]
    ds = ds.extension(:pagination)
    @paginated = ds.paginate(idx, count)
    return @paginated.all
  end

  def get_report_by_id(id)
    return @db[:ReportMetadata].filter(:id => id).all
  end

  def get_all_records_filtered(k,v)
    return @db[:Record].filter(k.to_sym => v).all
  end

  def get_records_by_report_id(id)
    return @db[:Record].filter(:report_metadata_id => id).all
  end

end
