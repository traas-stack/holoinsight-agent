#!/usr/bin/env bash
set -e

# 调用这个脚本在镜像里构建出 agent bin

script_dir=` dirname $0 `
project_root=`realpath $script_dir/../..`

echo '[build agent bin using docker]'

rm -rf $project_root/build/linux-amd64/bin

$script_dir/agent-builder/build.sh

# 在容器里打出agent bin
# docker build 时也使用宿主机网络, 避免nat故障时容器连不上网

builder_image=holoinsight-agent-builder:` cat $script_dir/agent-builder/VERSION `

echo '[build agent bin using docker]' docker run --network host --platform=linux/amd64 --rm -v $project_root:/a -v $HOME/.cache/go-build:/root/.cache/go-build $builder_image bash -c " sh /a/scripts/build/build-in-container.sh "
docker run --network host --platform=linux/amd64 --rm -v $project_root:/a -v $HOME/.cache/go-build:/root/.cache/go-build $builder_image bash -c " sh /a/scripts/build/build-in-container.sh "
