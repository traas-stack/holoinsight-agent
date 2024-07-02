#!/usr/bin/env bash
set -e

# VM 模式下的打包: 打成 tar 包, 用户下载之后解压启动即可

script_dir=`dirname $0 | xargs realpath`
project_root=`realpath $script_dir/../..`

version=` cat $project_root/VERSION `

echo '[build vm lite agent package]'

# tar 包目录结构:
# /bin
#   /agent agent 本体
# /data
#   /agent.yaml 配置文件, install 时生成

echo script dir is $script_dir
echo project root is $project_root

tmpdir=`mktemp -d`
agent_home=$tmpdir/agent

mkdir -p $agent_home/{bin,data,conf,logs}

echo temp agent home $agent_home

$script_dir/../build/build-using-docker.sh

cp $script_dir/../../build/linux-amd64/bin/agent $agent_home/bin/agent

echo
echo ls -lh $agent_home
ls -lh $agent_home
echo

build_target=$project_root/build/linux-amd64/holoinsight-agent-lite_linux-amd64_${version}.tar.gz
echo "build to $build_target"
cd $tmpdir/agent && tar -zcf $build_target *

echo you should upload $build_target to your OSS
