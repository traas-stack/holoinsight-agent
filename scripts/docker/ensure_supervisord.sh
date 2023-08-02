#!/usr/bin/env bash
set -e

if ! ps aux | grep /usr/bin/supervisord | grep -v grep >/dev/null 2>&1; then
  # Run as root with current envs visible
  sudo -E supervisord
  sleep 1
  pid_file=/var/run/supervisord.pid
  pid=`cat $pid_file`
  # see https://linux.die.net/man/5/proc
  sudo bash -c "echo -17 > /proc/$pid/oom_adj"
fi
