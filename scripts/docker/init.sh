#!/usr/bin/env bash
set -e

f=/usr/local/holoinsight/agent/bin/init_bashrc.sh
echo "[ -e $f ] && source $f" >> /root/.bashrc

rm /usr/local/holoinsight/agent/bin/init.sh
