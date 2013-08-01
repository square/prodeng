#!/usr/bin/ruby

$:.push '../lib'

require 'test/unit'
require 'elvis'


class TestElvis < Test::Unit::TestCase
 
  def test_elvis
    assert_nothing_raised do
#      Elvis.run_as('nobody') { } ## nobody known to fail on OSX/ruby 1.8.7 due to improper uid_t size handling in ruby core
      Elvis.run_as('daemon') { }
    end
    assert_raise ArgumentError do
      Elvis.run_as('abcdef-fakeuser') { }
    end
  end
end

