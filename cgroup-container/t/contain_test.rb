#!/usr/bin/ruby20

require 'test/unit'

class TestContainExecutable < Test::Unit::TestCase
   
   @@contain_cmd="./root/usr/bin/contain -d ./root/etc/container.d"

   def remove_cgroup
    system("cgdelete cpu:/test_contain_app >/dev/null 2>&1")
    system("cgdelete memory:/test_contain_app >/dev/null 2>&1")
   end

   def test_exec
     assert_equal(true, system("#{@@contain_cmd} -h >/dev/null 2>&1"))
   end

   def test_newcgroup_all
     Dir.entries("./root/etc/container.d").each do |dirent|
       next if dirent == '.'
       next if dirent == '..'
       dirent = File.basename(dirent, File.extname(dirent))
       remove_cgroup
       assert_equal(true, system(
         "#{@@contain_cmd} -s #{dirent} -a test_contain_app /bin/true"))
     end
   end

   # test if we set an existing cgroup from higher limits to lower
   def test_flip_small_tiny
     remove_cgroup
     # set test_contain_app to small
     assert_equal(true, system(
         "#{@@contain_cmd} -s small -a test_contain_app /bin/true"))
     # set test_contain_app to tiny
     assert_equal(true, system(
         "#{@@contain_cmd} -s tiny -a test_contain_app /bin/true"))
   end

   # test if we set an existing cgroup from lower limits to higher
   def test_flip_tiny_small
     remove_cgroup
     # set test_contain_app to tiny
     assert_equal(true, system(
         "#{@@contain_cmd} -s tiny -a test_contain_app /bin/true"))
     # set test_contain_app to small
     assert_equal(true, system(
         "#{@@contain_cmd} -s small -a test_contain_app /bin/true"))
   end
end
