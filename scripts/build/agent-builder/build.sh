#!/usr/bin/env bash

# doc: Run this script to build agent-builder image for multi arch.

set -e
script_dir=`dirname $0`

tag=$1

if [ -z "$tag" ]; then
  echo 'usage: build.sh <tag>'
  exit 1
fi

PLATFORM=linux/amd64 $script_dir/buildx-one.sh $tag-amd64-linux
PLATFORM=linux/arm64/v8 $script_dir/buildx-one.sh $tag-arm64v8-linux
