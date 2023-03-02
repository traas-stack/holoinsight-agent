#!/usr/bin/env bash
set -e

# doc: Run this script to detect which buildx to use in current machine.

buildx_bin=
if docker buildx >/dev/null 2>&1; then
  buildx_bin='docker buildx'
  exit 0
else
  # see https://docs.docker.com/build/install-buildx/
  tochecks=( "$HOME/.docker/cli-plugins" /usr/local/lib/docker/cli-plugins /usr/local/libexec/docker/cli-plugins /usr/lib/docker/cli-plugins /usr/libexec/docker/cli-plugins )

  for tocheck in ${tochecks[@]}; do
    if [ -e "$tocheck/buildx" ]; then
      buildx_bin="$tocheck/buildx"
      break
    fi
  done
fi

if [ -z "$buildx_bin" ]; then
  echo 'buildx not found'
  exit 1
fi

$buildx_bin $DOCKER_OTPS "$@"
