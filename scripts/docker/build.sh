#!/usr/bin/env bash
set -e

image=$1

script_dir=` dirname $0 `

project_root=`realpath $script_dir/../..`

# 在容器里打出镜像 或者 直接在本地构建
$script_dir/../build/build-bin-using-docker.sh

echo '[build agent docker image]'
docker build --network host --platform=linux/amd64 -f $script_dir/Dockerfile -t holoinsight_agent $project_root

echo you should upload holoinsight_agent to your public repository.
