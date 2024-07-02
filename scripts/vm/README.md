# 介绍
VM模式下的相关脚本.  
该脚本可以在 Mac m1 或 Linux 上执行, 要求机器上装有 docker. Mac m1 特别慢, 建议在 Linux 上执行, 可以通过 rsync 将代码复制到 Linux 上再构建.

# 使用
```bash
./scripts/vm/build.sh 1.0.0
```

构建的结果在 `/root/workspace/remote/cloudmonitor-agent/scripts/vm/holoinsight-agent_linux-amd64_1.0.0.tar.gz`

之后我们把它上传到 OSS, 给用户下载安装即可.  
```bash
./scripts/vm/upload.sh 1.0.0
```
