#!/usr/bin/env bash
set -e

# doc: Run this script to create and push docker manifest for specified arch.

tag=$1
if [ -z "$tag" ]; then
  echo 'usage: push-manifest.sh <tag>'
  exit 1
fi

docker manifest create holoinsight/agent-builder:$tag \
  -a holoinsight/agent-builder:$tag-amd64-linux \
  -a holoinsight/agent-builder:$tag-arm64v8-linux

docker manifest push holoinsight/agent-builder:$tag
