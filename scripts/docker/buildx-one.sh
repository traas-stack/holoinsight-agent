#!/usr/bin/env bash
set -e

# doc: Run this script to build agent image for specified arch.

image=$1

script_dir=` dirname $0 `

project_root=`realpath $script_dir/../..`

tag=$1
echo [$GOOS/$GOARCH] build agent bin
$script_dir/../build/build-using-docker.sh

buildx_bin=~/.docker/cli-plugins/buildx

echo [$GOOS/$GOARCH] [$PLATFORM] 'build agent docker image'
target_image=holoinsight/agent:$tag
echo $buildx_bin build --platform $PLATFORM -f $script_dir/Dockerfile -t $target_image $project_root
$buildx_bin build $DOCKER_OPTS --platform $PLATFORM -f $script_dir/Dockerfile -t $target_image $project_root

echo $target_image `docker inspect $target_image | jq ' .[0].Architecture '`
