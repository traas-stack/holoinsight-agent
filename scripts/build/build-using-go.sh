#!/usr/bin/env bash
set -e

# doc: Run this script to build agent binaries using go.

if [ -z "$GOOS" ] || [ -z "$GOARCH" ]; then
  GOOS=`go env GOOS`
  GOARCH=`go env GOARCH`
fi

# For cn user, use goproxy to speedup build
#go env -w GO111MODULE=on
#go env -w GOPROXY=https://goproxy.cn,direct

echo [$GOOS/$GOARCH] 'build agent bin using go'

script_dir=`dirname $0`
project_root=`realpath $script_dir/../..`
$project_root/scripts/gen-git-info.sh

version=`cat $project_root/VERSION`
buildTime=`TZ='Asia/Shanghai' date +'%Y-%m-%dT%H:%M:%S_%Z'`
gitcommit=`cat $project_root/gitcommit 2>/dev/null || true`
echo version=$version
echo buildTime=$buildTime
echo gitcommit=$gitcommit

# https://www.digitalocean.com/community/tutorials/using-ldflags-to-set-version-information-for-go-applications
(cd $project_root && \
  CGO_ENABLED=1 go build \
  -ldflags "-s -w" \
  -ldflags "-X github.com/traas-stack/holoinsight-agent/pkg/appconfig.agentVersion=$version \
  -X github.com/traas-stack/holoinsight-agent/pkg/appconfig.agentBuildTime=$buildTime \
  -X github.com/traas-stack/holoinsight-agent/pkg/appconfig.gitcommit=$gitcommit" \
  -o build/$GOOS-$GOARCH/bin/agent ./cmd/agent && \
  CGO_ENABLED=0 go build -ldflags "-s -w" -o build/$GOOS-$GOARCH/bin/helper ./cmd/containerhelper)
