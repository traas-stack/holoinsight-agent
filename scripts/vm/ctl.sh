#!/usr/bin/env bash
set -e

bin_dir=`dirname $0`
bin_dir=`cd $bin_dir && echo $PWD`
echo bin_dir is $bin_dir

SUPERVISORD_BIN=$bin_dir/supervisord
SUPERVISORD_CONF=${SUPERVISORD_BIN}.conf
SUPERVISORD_PID_FILE=$bin_dir/../run/supervisord.pid


function start() {
  # TODO 防止重复启动
  if [ -e "$SUPERVISORD_PID_FILE" ]; then
    pid=`cat $SUPERVISORD_PID_FILE`
    if ps -p $pid >/dev/null; then
      # PID 存在
      return
    else
      rm -f $SUPERVISORD_PID_FILE
      # 文件存在但pid不存在, 启动
      $SUPERVISORD_BIN -c $SUPERVISORD_CONF -d
    fi
  else
    # pid文件不存在, 启动
    $SUPERVISORD_BIN -c $SUPERVISORD_CONF -d
  fi

  $SUPERVISORD_BIN -c $SUPERVISORD_CONF ctl start agent
}

function stop() {
  $SUPERVISORD_BIN -c $SUPERVISORD_CONF ctl stop agent

  out=`$SUPERVISORD_BIN -c $SUPERVISORD_CONF ctl shutdown`
  if [ "Shut Down" != "$out" ] && [ "Hmmm! Something gone wrong?!" != "$out" ]; then
    echo "stop supervisord: $out"
  fi

  if [ -e "$SUPERVISORD_PID_FILE" ]; then
    pid=`cat $SUPERVISORD_PID_FILE`
    if ps -p $pid >/dev/null 2>&1; then
      kill $pid || true
    fi
    rm $SUPERVISORD_PID_FILE
  fi

  pid=`ps aux | grep $SUPERVISORD_BIN | grep -v grep | awk '{ print $2 }'`
  if [ -n "$pid" ]; then
    kill $pid || true
  fi
}

function restart() {
  $SUPERVISORD_BIN -c $SUPERVISORD_CONF ctl restart agent
}

function status() {
  $SUPERVISORD_BIN -c $SUPERVISORD_CONF ctl status agent
}

case "$1" in
start)
  start
;;
stop)
  stop
;;
restart)
  restart
;;
status)
  status
;;
*)
  echo 'usage: ctl.sh <start|stop|restart|status>'
  exit 1
;;
esac

