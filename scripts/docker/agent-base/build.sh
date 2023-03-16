#!/usr/bin/env bash
set -e

# docs: Run thi script to build base image for agent and push images to Docker Hub

cd `dirname $0`

tag=$1
if [ -z "$tag" ]; then
  echo 'usage: build.sh <tag>'
  exit 1
fi


docker buildx build \
  --platform=linux/amd64,linux/arm64/v8 \
  -t holoinsight/agent-base:$tag \
  --pull --push .
