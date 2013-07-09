require 'net/imap'
require 'mail'
require 'zipruby'

module DMARC
  class Report
    class Retrieve
      class IMAP
      end
    end
  end
end

class DMARC::Report::Retrieve::IMAP
  def initialize(args)
    folder = args[:folder] || 'INBOX'
    server = args[:server] || 'server'
    port   = args[:port]   ||  993
    ssl    = args[:ssl]    ||  false
    imap = Net::IMAP.new(server, port, ssl)
    imap.login(user, password)
    imap.select(folder)
  end

  def fetch_aggregate_reports(mark_seen)
    result = []
    imap.search(["UNSEEN"]).each do |message_id|
      body = imap.fetch(message_id, "BODY[]")[0].attr["BODY[]"]
      mail = Mail.new(body)
      mail.attachments.each do |a|
        Zip::Archive.open_buffer(a.body.decoded) do |zf|
          zf.fopen(zf.get_name(0)) do |f|
            m = DMARC::Report::Aggregate.new
            m.parse(f.read)
            if(block_given?)
              yield m
            else
              result.push(m)
            end
          end
        end
      end
      if(mark_seen)
        imap.store(message_id, "+FLAGS", [:Seen])
      end
    end
    return result
  end

  def finalize()
    imap.logout()
    imap.disconnect()
  end

end
