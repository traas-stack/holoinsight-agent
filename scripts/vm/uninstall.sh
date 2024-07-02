#!/usr/bin/env bash
set -e

bin_dir=`dirname $0 | xargs realpath`
echo bin dir $bin_dir

agent_home=` realpath $bin_dir/.. `
echo agent home $agent_home

if [ -e "$bin_dir/ctl.sh" ]; then
  echo "find $bin_dir/ctl.sh, $bin_dir/ctl.sh stop first"
  $bin_dir/ctl.sh stop
fi

rm -rf $agent_home
