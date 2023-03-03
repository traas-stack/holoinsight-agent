#!/usr/bin/env bash
set -e

# doc: Run this script to build agent binaries using docker

script_dir=` dirname $0 `
project_root=`realpath $script_dir/../..`

if [ -z "$GOOS" ] || [ -z "$GOARCH" ] || [ -z "$PLATFORM" ]; then
  echo 'require env: GOOS/GOARCH/PLATFORM'
  exit 1
fi

echo [$GOOS/$GOARCH] [$PLATFORM] 'build agent bin using docker'
rm -rf $project_root/build/$GOOS-$GOARCH/bin

builder_image=holoinsight/agent-builder:1.0.0

docker run \
  $DOCKER_OPTS \
  --platform $PLATFORM \
  -e GOOS=$GOOS \
  -e GOARCH=$GOARCH \
  --rm \
  -v $project_root:/workspace \
  -v $HOME/.cache/go-build:/root/.cache/go-build \
  -v $HOME/go/pkg:/root/go/pkg \
  $builder_image /workspace/scripts/build/build-in-container.sh
