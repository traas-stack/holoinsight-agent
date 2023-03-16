#!/usr/bin/env bash

# doc: Run this script to build agent-builder image for multi arch.

cd `dirname $0`

set -e
script_dir=`dirname $0`

tag=$1

if [ -z "$tag" ]; then
  echo 'usage: build.sh <tag>'
  exit 1
fi

docker buildx build \
  --platform linux/amd64,linux/arm64/v8 \
  -t holoinsight/agent-builder:$tag \
  --pull --push .
