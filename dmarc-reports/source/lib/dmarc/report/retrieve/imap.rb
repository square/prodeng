require 'net/imap'
module DMARC
  class Report
    class Retrieve
      class IMAP
      end
    end
  end
end

class DMARC::Report::Retrieve::IMAP
  def initialize(server,username,password,folder)
    folder ||= 'INBOX'
    imap = Net::IMAP.new(server, 993, true)
    imap.login(user, password)
    imap.select(folder)
  end

  def fetch(mark_seen)
    imap.search(["NOT", "SEEN"]).each do |message_id|
    end
    if(mark_seen)
      imap.store(message_id, "+FLAGS", [:Seen])
    end
  end

  def finalize()
    imap.logout()
    imap.disconnect()
  end
end
