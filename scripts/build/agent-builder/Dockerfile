FROM centos:7
# 这是用于构建 agent 的基础镜像, 非 agent 运行时的镜像
# 我们不能直接使用 golang:1.17 因为它打出来的 bin 依赖的 GLIBC 版本太高, 导致 bin 无法在一些低版本的 OS 上运行
# 报错如下:
# /usr/local/holoinsight/agent/bin/agent: /lib64/libc.so.6: version `GLIBC_2.28' not found (required by /usr/local/holoinsight/agent/bin/agent)

# 安装 cgo 依赖的工具

RUN yum install -y make gcc && \
  yum -q -y clean all && \
  rm -rf /var/cache/yum

ADD http://mirrors.ustc.edu.cn/golang/go1.19.4.linux-amd64.tar.gz /opt

RUN cd /opt && tar xvf go1.19.4.linux-amd64.tar.gz >/dev/null

ENV GOROOT=/opt/go
ENV PATH=$PATH:$GOROOT/bin
