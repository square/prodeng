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

  def initialize(connstr)
    @db = Sequel.connect(connstr)
  end
  
  # summary[your senders|forwarded|unknown]
  #               count =    xxx
  #               spf_fail = xxx
  #               spf_pass = xxx
  #               dkim_pass = xxx
  #               dkim_fail = xxx
  #
  # drill by header_from or domain_name
  def summary(args)
    out = {'known'=> {}, 'forwarded'=>{},'unknown'=>{} }
    args[:end] ||= Time.now
    args[:start]   ||= args[:end] - 10*86400 # 10 days
    get_all_reports_by_date(args[:start],args[:end]).each do |r|
      get_all_records_filtered("report_metadata_id",r.id).each do |record|
        # if record.source_ip is part of authorized_senders
        out['known']['count'] += record.count
      end
    end
    return out
  end

  def get_all_reports(limit)
    if(limit)
      return @db[:ReportMetadata].limit(limit).all
    else
      return @db[:ReportMetadata].all
    end
  end

  def get_all_reports_by_date(s,e)
    return @db[:ReportMetadata].where{begin_date >= s && end_date <= e}
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
