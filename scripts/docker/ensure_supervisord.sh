#!/usr/bin/env bash
set -e

if ! ps aux | grep /usr/bin/supervisord | grep -v grep >/dev/null 2>&1; then
  # Run as root with current envs visible
  sudo -E supervisord
fi
