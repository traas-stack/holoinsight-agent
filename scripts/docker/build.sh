#!/usr/bin/env bash
set -e

# doc: Run this script to build agent image for multi arch.

image=$1

script_dir=` dirname $0 `
project_root=`realpath $script_dir/../..`

function buildOne() {
  local tag=$1

  #$script_dir/../build/build-using-go.sh
  $script_dir/../build/build-using-docker.sh

  echo [$GOOS/$GOARCH] [$PLATFORM] 'build agent docker image'
  local image=holoinsight/agent:$tag
  docker buildx build \
    $DOCKER_OPTS \
    --build-arg GOOS=$GOOS \
    --build-arg GOARCH=$GOARCH \
    --platform $PLATFORM \
    -f $script_dir/Dockerfile \
    -t $image \
    $project_root
  echo $image `docker inspect holoinsight/agent:$tag | jq ' .[0].Architecture '`
}

GOOS=linux GOARCH=amd64 PLATFORM=linux/amd64 buildOne test-amd64-linux
GOOS=linux GOARCH=arm64 PLATFORM=linux/arm64/v8 buildOne test-arm64v8-linux

echo Notice: you should upload holoinsight/agent to your public repository.
