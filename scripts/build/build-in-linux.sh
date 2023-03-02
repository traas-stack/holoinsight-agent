#!/usr/bin/env bash
set -e

# doc: Run this script in Linux to build agent binaries.

if [ `uname` != 'Linux' ]; then
  echo 'build-in-linux.sh must run in Linux OS'
  exit 1
fi

cd `dirname $0`/../..

./scripts/gen-git-info.

./scripts/build/build-using-go.sh
