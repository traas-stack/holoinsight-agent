#!/usr/bin/env bash
set -e

# 开机自启动时会调用这个脚本
exec /usr/local/holoinsight/agent/bin/ctl.sh "$@"
