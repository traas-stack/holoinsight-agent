#!/usr/bin/env bash
set -e

# 这个脚本是安装agent的脚本, 假设当前目录下有
# - install
# - holoinsight-agent_linux-amd64_latest.tar.gz

# usage: bash install -a ${应用名} -i ${apikey} -r ${Registry服务端地址} -g ${Gateway服务端地址} -t ${holoinsight-agent tar.gz 包的本地路径} -h ${指定一个agent的安装目录} -d ${如果安装时agent的安装目录已经存在, 是否先删除已存在的agent的安装目录}
# 参数解释:
# 参数	必填	解释
# -a	否	应用名
# -i	是	apikey
# -r	是	registry服务端地址
# -g	是	gateway服务端地址
# -t	是	agent安装包地址
# -h	否	用于指定agent的安装目录(即 agent_home ) 如果不填, 默认会安装在 ./agent，即当前目录下
# -d	否	如果 agent_home 已经存在, 是否先删除它

# 对于每个给定的环境而言, -r -g -t 是固定的
# 安装和运行 agent 不需要root权限, 除非你需要采集 root 权限的数据 ( 比如 /root 目录下的日志 ).
# agent 总是以当前用户(执行启动脚本的用户)执行.

agent_tar_url=
hi_app=
hi_apikey=
hi_registry_addr=
hi_gateway_addr=
hi_home=
# 如果为true则表示删除旧目录
hi_delete_old=
debug_grpc_secure=true

while getopts "a:i:r:g:t:h:ds:" OPT; do
  case "$OPT" in
  a)
      hi_app=$OPTARG
    ;;
  i)
    hi_apikey=$OPTARG
    ;;
  r)
    hi_registry_addr=$OPTARG
    ;;
  g)
    hi_gateway_addr=$OPTARG
    ;;
  t)
    agent_tar_url=$OPTARG
    ;;
  h)
    hi_home=$OPTARG
    ;;
  d)
    hi_delete_old=1
    ;;
  s)
    debug_grpc_secure=$OPTARG
    ;;
  *)
    echo "unknown opt $OPT"
    exit 1
    ;;
  esac
done

echo "install params:"
echo "hi_app=$hi_app"
echo "hi_apikey=$hi_apikey"
echo "hi_registry_addr=$hi_registry_addr"
echo "hi_gateway_addr=$hi_gateway_addr"
echo "hi_home=$hi_home"
echo "hi_delete_old=$hi_delete_old"
echo "debug_grpc_secure=$debug_grpc_secure"

echo

if [ -z "$hi_apikey" ]; then
  echo 'apikey is empty'
  exit 1
fi

if [ -z "$hi_registry_addr" ]; then
  echo 'registry addr is empty'
  exit 1
fi

if [ -z "$hi_gateway_addr" ]; then
  echo 'gateway addr is empty'
  exit 1
fi


if [ -z "$agent_tar_url" ]; then
  echo 'agent tar url is empty'
  exit 1
fi

# 以当前目录作为 $agent_home

cwd=$PWD
echo "current working directory $cwd"

if [ -n "$hi_home" ]; then
  agent_home=$hi_home
else
  agent_home=$PWD/agent
  echo "hi_home is empty, use $agent_home as agent home"
fi

echo "use agent home $agent_home"

# 先停止 (如果存在的话)
if [ -e "$agent_home/bin/ctl.sh" ]; then
  echo "find $agent_home/bin/ctl.sh, stop agent first"
  $agent_home/bin/ctl.sh stop
fi

if ps aux | grep $agent_home/bin | grep -v grep >/dev/null; then
  echo "kill processes related to $agent_home/bin"
  for pid in `ps aux | grep $agent_home/bin | grep -v grep | awk '{ print $2 }'`; do
    echo "$$ $pid"
    if [ "$$" != "$pid" ]; then
      echo kill $pid `cat /proc/$pid/cmdline | tr '\0' ' '`
      kill $pid
    fi
  done
fi

# 卸载 (如果存在的话)
if [ "$hi_delete_old" = "1" ] && [ -e "$agent_home" ]; then
  echo "remove directory $agent_home"
  rm -rf $agent_home
fi

# 开始安装

echo "holoinsight agent will be install on $agent_home"

# 入参会有一个url, 下载 tar.gz 包到临时目录
tmpdir=`mktemp -d`
echo "make temp dir $tmpdir"

echo "download agent tar from $agent_tar_url to $tmpdir"

if [ -e "$agent_tar_url" ]; then
  echo "use local agent tar /$agent_tar_url"
else
  if command -v wget >/dev/null; then
    wget -O $tmpdir/holoinsight-agent_linux-amd64.tar.gz $agent_tar_url
  elif command -v curl >/dev/null; then
    curl -o $tmpdir/holoinsight-agent_linux-amd64.tar.gz $agent_tar_url
  else
    echo "there is no 'wget' or 'curl' in PATH"
    exit 1
  fi
fi

# 解压到标准目录, 需要sudo (如何检查自己能否无密码sudo?)
mkdir -p $agent_home

echo "unarchive agent tar to $agent_home/.."
tar -zxf $tmpdir/holoinsight-agent_linux-amd64.tar.gz -C $agent_home

# 此时的目录结构 /usr/local/holoinsight/agent/...

# 修复权限
chmod a+x $agent_home/bin/*

# 初始化 agent.yaml
echo "apikey: \"$hi_apikey\"
app: \"$hi_app\"
registry:
  addr: $hi_registry_addr
  secure: $debug_grpc_secure
gateway:
  addr: $hi_gateway_addr
  secure: $debug_grpc_secure
" > ${agent_home}/conf/agent.yaml

echo
echo "conf/agent.yaml:"
cat ${agent_home}/conf/agent.yaml
echo

# 建立相关目录
mkdir -p $agent_home/{logs,run}

chmod a+x $agent_home/bin/initd_holoinsight-agent.sh

# 更新 supervisor 地址
sed -i s@/usr/local/holoinsight/agent@$agent_home@g $agent_home/bin/supervisord.conf
sed -i s@/usr/local/holoinsight/agent@$agent_home@g $agent_home/bin/agent.ini

# 启动agent
echo "start agent"
$agent_home/bin/ctl.sh start

sleep 1

echo "agent status"
$agent_home/bin/ctl.sh status
