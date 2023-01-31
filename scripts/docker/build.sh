#!/usr/bin/env bash
set -e

image=$1

script_dir=` dirname $0 `

project_root=`realpath $script_dir/../..`

$script_dir/../build/build-using-docker.sh

echo '[build agent docker image]'
docker build --network host --platform=linux/amd64 -f $script_dir/Dockerfile -t holoinsight/agent $project_root

echo Notice: you should upload holoinsight/agent to your public repository.
