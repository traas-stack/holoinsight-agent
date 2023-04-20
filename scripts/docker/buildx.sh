#!/usr/bin/env bash
set -e

# docs: Build agent docker image for multi arch and push images to Docker Hub

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
./scripts/gen-git-info.sh

docker buildx build \
  --platform linux/amd64,linux/arm64/v8 \
  $build_opts \
  -f ./scripts/docker/Dockerfile \
  --pull --push \
  -t holoinsight/agent:$tag .
