FROM golang:1.22 as golang

FROM centos:7
# This image is used to build agent bin. We use old version centos:7 to avoid GLIBC error in some older version OS.
# GLIBC error example:
# /usr/local/holoinsight/agent/bin/agent: /lib64/libc.so.6: version `GLIBC_2.28' not found (required by /usr/local/holoinsight/agent/bin/agent)

RUN yum install -y make gcc && \
  yum -q -y clean all && \
  rm -rf /var/cache/yum

COPY --from=golang /usr/local/go /usr/local/go

ENV GOROOT=/usr/local/go
ENV PATH=$PATH:$GOROOT/bin
