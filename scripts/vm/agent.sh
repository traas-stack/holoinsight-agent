#!/usr/bin/env bash

export GOTRACEBACK=all
export GODEBUG=gctrace=1,madvdontneed=1

script_dir=`dirname $0`

# 如果文件存在就从它加载一些环境变量
if [ -e "$script_dir/env.sh" ]; then
  source $script_dir/env.sh
fi

exec $script_dir/agent

