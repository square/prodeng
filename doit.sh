#!/bin/bash
# ugly hack to build/test multiple projects in one repo

set -x
set -e

for i in */; do
  if [ -e "${i}doit.sh" ]; then
    echo "BUILDING PROJECT $i"
    pushd .
    cd "$i"
    "./doit.sh"
    popd
  fi
done
