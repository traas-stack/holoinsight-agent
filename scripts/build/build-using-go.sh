#!/usr/bin/env bash
set -e

script_dir=`dirname $0`
project_root=`realpath $script_dir/../..`

os=`go env GOOS`
arch=`go env GOARCH`

go env

version=`cat $project_root/VERSION`
buildTime=`TZ='Asia/Shanghai' date +'%Y-%m-%dT%H:%M:%S_%Z'`
gitcommit=`cat $project_root/gitcommit`
echo version=$version
echo buildTime=$buildTime
echo gitcommit=$gitcommit

# https://www.digitalocean.com/community/tutorials/using-ldflags-to-set-version-information-for-go-applications
(cd $project_root && \
  go build \
  -ldflags "-s -w" \
  -ldflags "-X github.com/TRaaSStack/holoinsight-agent/pkg/appconfig.agentVersion=$version \
  -X github.com/TRaaSStack/holoinsight-agent/pkg/appconfig.agentBuildTime=$buildTime \
  -X github.com/TRaaSStack/holoinsight-agent/pkg/appconfig.gitcommit=$gitcommit" \
  -o build/$os-$arch/bin/agent ./cmd/agent && \
  go build -ldflags "-s -w" -o build/$os-$arch/bin/helper ./cmd/containerhelper)
