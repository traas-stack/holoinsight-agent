#!/usr/bin/env bash
set -e

# 普通的upload 仅 upload 到 snapshot 版本, 防止开发人员意外操作覆盖线上版本

script_dir=`dirname $0 | xargs realpath`
project_root=`realpath $script_dir/../..`

version=` cat $project_root/VERSION `

echo "version is $version"

file="$project_root/build/linux-amd64/holoinsight-agent_linux-amd64_${version}.tar.gz"
tar=`basename $file`

# 运行手动export
#ak=``
#sk=``
#ossHost=``
#ossBucket=``

# 上传安装脚本
echo '[snapshot][upload install.sh and package to OSS]'
ossutil -e $ossHost -i $ak -k $sk cp -f $script_dir/install.sh oss://$ossBucket/agent/install-snapshot
echo "[snapshot]upload $file to oss"

ossutil -e $ossHost -i $ak -k $sk cp -f $file oss://$ossBucket/agent/holoinsight-agent_linux-amd64_snapshot.tar.gz
echo "[snapshot]you can download from oss"
