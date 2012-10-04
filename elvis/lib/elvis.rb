#!/usr/bin/ruby

require 'etc'

module Elvis
  
  def Elvis.check_threads
    if Thread.list.length != 1
      raise "Elvis.run_as cannot be used with threads"
    end
  end
  def Elvis.verify_drop_privs(newuid, newgid)
    raise "could not set real gid" unless newgid == Process.gid
    raise "could not set effective gid" unless newgid == Process.egid
    raise "could not set real uid" unless newuid == Process.uid
    raise "could not set effective uid" unless newuid == Process.euid
  end
  def Elvis.verify_restore_privs(olduid, oldgid)
    raise "could not set real gid" unless oldgid == Process.gid
    raise "could not set effective gid" unless oldgid == Process.egid
    raise "could not set real uid" unless olduid == Process.uid
    raise "could not set effective uid" unless olduid == Process.euid
  end
  arch, os = RUBY_PLATFORM.split('-')
  if os =~ /^darwin/
    def Elvis.run_as(user)
      check_threads
      pw = Etc.getpwnam(user)
      
      original_uid = Process.euid
      original_gid = Process.egid
      original_groups = Process.groups
      begin
        Process::Sys.setregid(pw.gid, pw.gid)
        Process.initgroups(user, pw.gid)
        Process::Sys.seteuid(pw.uid)
        Process::Sys.setreuid(pw.uid, -1)
        Elvis.verify_drop_privs(pw.uid, pw.gid)

        yield
      ensure
        Process::Sys.setreuid(original_uid, original_uid)
        Process::Sys.setregid(original_gid, original_gid)
        Process.groups = original_groups
        Elvis.verify_restore_privs(original_uid, original_gid)
      end
    end
  elsif os == "linux"
    def Elvis.run_as(user)
      check_threads
      pw = Etc.getpwnam(user)
      
      original_uid = Process.euid
      original_gid = Process.egid
      original_groups = Process.groups
      begin
        Process::Sys.setresgid(pw.gid, pw.gid, -1)
        Process.initgroups(user, pw.gid)
        Process::Sys.setresuid(pw.uid, pw.uid, -1)
        Elvis.verify_drop_privs(pw.uid,pw.gid)
        
        yield
      ensure
        Process::Sys.setresuid(original_uid, original_uid, -1)
        Process::Sys.setresgid(original_gid, original_gid, -1)
        Process.groups = original_groups
        Elvis.verify_restore_privs(original_uid, original_gid)
      end
    end
  else
    raise "unknown platform: #{RUBY_PLATFORM}. set*id() functions do not have standardized behavior between systems."
  end
end

if __FILE__ == $0
  print "Starting up as:   "
  system "id"
  begin
    Elvis.run_as(ARGV.first) {
      print "inside block as:  "
      system "id"
    }
  ensure
    print "outside block as: "
    system "id"
  end
end
