#!/usr/bin/ruby
# local tests which don't require a range server

$:.push '../lib'

require 'test/unit'
require 'rangeclient'


class TestRangeClient < Test::Unit::TestCase
 
  def test_compress
    r = Range::Client.new
    assert_equal(r.compress(%W{foo100 foo101}), "foo100..1")
  end
 
end

