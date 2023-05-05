#!/usr/bin/env bash
set -e

if [ -n "$GOPROXY" ]; then
  echo use GOPROXY=$GOPROXY
elif go env GOPROXY >/dev/null 2>&1; then
  export GOPROXY=`go env GOPROXY`
  echo use GOPROXY=$GOPROXY
fi
