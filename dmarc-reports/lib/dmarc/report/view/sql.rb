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

  def get_all_reports()
    return @db[:ReportMetadata].all
  end

  def get_all_reports_paginated(idx,n)
    ds = @db[:ReportMetadata]
    ds = ds.extension(:pagination)
    @paginated = ds.paginate(idx, n)
    return @paginated
  end

  def get_report_by_id(id)
    return @db[:ReportMetadata].filter(:id => id).all
  end

  def get_records_by_report_id(id)
    return @db[:Record].filter(:report_metadata_id => id).all
  end

end
