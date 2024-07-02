#!/usr/bin/env bash
set -e

# 这个脚本发版本到线上, 必须非常小心

version="$1"
if [ -z "$version" ]; then
  echo 'usage: upload.sh <version>'
  exit 1
fi

script_dir=`dirname $0 | xargs realpath`
project_root=`realpath $script_dir/../..`

echo "version is $version"

file="$project_root/build/linux-amd64/holoinsight-agent_linux-amd64_${version}.tar.gz"
tar=`basename $file`

# 运行手动export
#ak=``
#sk=``
#ossUrl=``
#ossBucket=``

# 上传安装脚本
echo '[upload install.sh and package to OSS]'
ossutil -e $ossUrl -i $ak -k $sk cp -f $script_dir/install.sh oss://$ossBucket/agent/install
echo "upload $file to oss"

oss_target="oss://$ossBucket/agent/$tar"

# 检查一下文件是否存在, 如果存在别随便覆盖!!!
echo stat $oss_target

# 如果为true则表示要强制覆盖, 这个比较危险, 对于已经打包的, 我们最好别覆盖了
OVERWRITE="0"

if ! ossutil -e $ossUrl -i $ak -k $sk stat $oss_target; then
  ossutil -e $ossUrl -i $ak -k $sk cp -f $file $oss_target
elif [ "$OVERWRITE" = "1" ]; then
  echo "warning: overwrite $oss_target already exists"
  ossutil -e $ossUrl -i $ak -k $sk cp -f $file $oss_target
else
  # 多打印几行 引起注意
  for i in `seq 10`; do
    echo "warning: $oss_target already exists"
  done
  exit 1
fi

ossutil -e $ossUrl -i $ak -k $sk cp -f $file oss://$ossBucket/agent/$tar
ossutil -e $ossUrl -i $ak -k $sk cp -f $file oss://$ossBucket/agent/holoinsight-agent_linux-amd64_latest.tar.gz
echo "you can download from oss"
