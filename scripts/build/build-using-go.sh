#!/usr/bin/env bash
set -e

# doc: Run this script to build agent binaries using go.

if [ -z "$GOOS" ] || [ -z "$GOARCH" ]; then
  echo 'require env: GOOS/GOARCH'
  exit 1
fi

echo [$GOOS/$GOARCH] 'build agent bin using go'

script_dir=`dirname $0`
project_root=`realpath $script_dir/../..`

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
  -ldflags "-X github.com/traas-stack/holoinsight-agent/pkg/appconfig.agentVersion=$version \
  -X github.com/traas-stack/holoinsight-agent/pkg/appconfig.agentBuildTime=$buildTime \
  -X github.com/traas-stack/holoinsight-agent/pkg/appconfig.gitcommit=$gitcommit" \
  -o build/$GOOS-$GOARCH/bin/agent ./cmd/agent && \
  go build -ldflags "-s -w" -o build/$GOOS-$GOARCH/bin/helper ./cmd/containerhelper)
