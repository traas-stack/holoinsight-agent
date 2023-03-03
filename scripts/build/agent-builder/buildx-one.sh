#!/usr/bin/env bash
set -e

# doc: Run this script to build agent-builder image for specified arch.

script_dir=`dirname $0`
tag=$1

if [ -z "$PLATFORM" ] || [ -z "$tag" ]; then
  echo 'usage: PLATFORM=<platform> buildx-one.sh <tag>'
  exit 1
fi

echo [$PLATFORM] build agent-builder iamge $tag
image=holoinsight/agent-builder:$tag

buildx_bin=$script_dir/../../docker/buildx.sh

$buildx_bin build $DOCKER_OPTS --platform=$PLATFORM -t $image -f $script_dir/Dockerfile $script_dir
