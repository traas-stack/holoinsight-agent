#!/usr/bin/env bash
set -e

script_dir=`dirname $0`
project_root=`realpath $script_dir/..`

if command -v git >/dev/null 2>&1 && [ -e ".git" ]; then
  echo '[gen-git-info] gen git info'
  git rev-parse HEAD > $project_root/gitcommit
else
  echo '[gen-git-info] no git command or .git'
  echo "unknown" > $project_root/gitcommit
fi
