#!/usr/bin/env bash
set -e
script_dir=`dirname $0`
version=` cat $script_dir/VERSION `

image=holoinsight-agent-builder:$version

if docker images $image | grep $version >/dev/null 2>&1; then
    # image exists, exit
    echo "[holoinsight-agent-builder] builder image [$image] already exists"
    exit 0
fi

echo "[holoinsight-agent-builder] build agent builder image=[$image]"

docker build --network host --platform=linux/amd64 -t $image -f $script_dir/Dockerfile $script_dir

docker tag holoinsight-agent-builder:$version holoinsight-agent-builder:latest
