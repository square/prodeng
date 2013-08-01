#!/bin/sh

gem build elvis.gemspec

cd t
for i in *.t; do echo "Testing: $i"; sudo ./$i || exit 1; done

