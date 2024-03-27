#!/usr/bin/env bash
set -e

# doc: Run this script to build agent binaries using docker

script_dir=` dirname $0 `
project_root=`realpath $script_dir/../..`

echo 'build agent bin using docker'
GOOS=`go env GOOS`
GOARCH=`go env GOARCH`
rm -rf $project_root/build/$GOOS-$GOARCH/bin

builder_image=holoinsight/agent-builder:1.0.2

mkdir -p $HOME/.cache/go-build $HOME/go/pkg

# If user defines GOPROXY env, then pass it to docker build using --build-arg.
build_opts=""
source $script_dir/detect-goproxy.sh
build_opts=""
if [ -n "$GOPROXY" ]; then
  build_opts="-e GOPROXY=$GOPROXY"
fi

docker run \
  --rm \
  $build_opts \
  -v $project_root:/workspace \
  -v $HOME/.cache/go-build:/root/.cache/go-build \
  -v $HOME/go/pkg:/root/go/pkg \
  $builder_image /workspace/scripts/build/build-using-go.sh
