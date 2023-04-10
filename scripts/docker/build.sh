#!/usr/bin/env bash
set -e

# docs: Build agent docker image for current arch locally

cd `dirname $0`/../..

if [ -z "$tag" ]; then
  tag=latest
fi

dockerfile=./scripts/docker/Dockerfile

if [ "$local"x = "1"x ]; then
  # local build is quick but not recommended, because it may have dependencies on local env
  ./scripts/build/build-using-go.sh
  dockerfile=${dockerfile}-local
else
  # If user defines GOPROXY env, then pass it to docker build using --build-arg.
  build_opts=""
  source ./scripts/build/detect-goproxy.sh
  build_opts=""
  if [ -n "$GOPROXY" ]; then
    build_opts="--build-arg GOPROXY=$GOPROXY"
  fi
fi

docker buildx build \
    $build_opts \
    -f $dockerfile \
    --load \
    -t holoinsight/agent:$tag .
