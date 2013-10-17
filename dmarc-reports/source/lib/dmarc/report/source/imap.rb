require 'net/imap'
require 'mail'
require 'zipruby'

module DMARC
  class Report
    class Source
      class IMAP
      end
    end
  end
end

class DMARC::Report::Source::IMAP
  def initialize(args)
    folder = args['folder'] || 'INBOX'
    server = args['server'] || 'server'
    port   = args['port']   ||  993
    ssl    = args['ssl']    ||  false
    username   = args['username']
    password  = args['password']
    @imap = Net::IMAP.new(server, port, ssl)
    @imap.login(username, password)
    @imap.select(folder)
  end

  def fetch_aggregate_reports(args)
    args ||= {}
    args[:mark_seen] ||= true
    result = []
    @imap.search(["UNSEEN"]).each do |message_id|
      body = @imap.fetch(message_id, "BODY[]")[0].attr["BODY[]"]
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

      if(args[:mark_seen])
        @imap.store(message_id, "+FLAGS", [:Seen])
      else
        @imap.store(message_id, "-FLAGS", [:SEEN])
      end

    end
    return result
  end

  def finalize()
    @imap.logout()
    @imap.disconnect()
  end

end
