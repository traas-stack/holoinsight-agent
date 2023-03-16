#!/usr/bin/env bash
set -e

# docs: Build agent docker image for multi arch and push images to Docker Hub

cd `dirname $0`/../..

if [ -z "$tag" ]; then
  tag=latest
fi

docker buildx build \
  --platform linux/amd64,linux/arm64/v8 \
  -f ./scripts/docker/Dockerfile \
  --pull --push \
  -t holoinsight/agent:$tag .
