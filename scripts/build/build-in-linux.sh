#!/usr/bin/env bash
set -e
# 该脚本只能在linux环境执行

if [ `uname` != 'Linux' ]; then
  echo 'build-in-linux.sh must run in Linux OS'
  exit 1
fi

script_dir=` dirname $0 `

project_root=`realpath $script_dir/../..`

cd $project_root && sh $script_dir/../gen-git-info.sh && $script_dir/build-using-go.sh
