#!/usr/bin/ruby

$:.push '../lib'

require 'test/unit'
require 'io/poll'


class TestIOPoll < Test::Unit::TestCase
 
  def test_select
    ret = IO.select_using_poll([STDIN], [STDOUT], [], 5)
    assert_equal(ret, [[],[STDOUT],[]])
  end
 
end

