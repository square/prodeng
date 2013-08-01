#!/bin/sh

gem build io-poll.gemspec

cd t
for i in *.t; do echo "Testing: $i"; ./$i || exit 1; done

