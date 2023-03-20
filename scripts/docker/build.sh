#!/usr/bin/env bash
set -e

# docs: Build agent docker image for current arch locally

cd `dirname $0`/../..

if [ -z "$tag" ]; then
  tag=latest
fi

# If user defines GOPROXY env, then pass it to docker build using --build-arg.
build_opts=""
source ./scripts/build/detect-goproxy.sh
build_opts=""
if [ -n "$GOPROXY" ]; then
  build_opts="--build-arg GOPROXY=$GOPROXY"
fi

docker buildx build \
  --platform linux/amd64 \
  $build_opts \
  -f ./scripts/docker/Dockerfile \
  --load \
  -t holoinsight/agent:$tag .
