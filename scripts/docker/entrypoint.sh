#!/usr/bin/env bash

if [ "$USESUPERVISOR" = "true" ]; then
  echo 'use supervisord to manage agent process'
  exec /usr/bin/supervisord -n
fi

# TODO stdout log rotate
exec >>/usr/local/holoinsight/agent/logs/stdout.log
exec 2>&1

exec /usr/local/holoinsight/agent/bin/app.sh
