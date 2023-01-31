#!/usr/bin/env bash
set -e

if ! ps aux | grep /usr/bin/supervisord | grep -v grep >/dev/null 2>&1; then
  # 进程不存在, 先启动一下
  return
fi

sc stop app
