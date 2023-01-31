FROM centos:7

# 为了独立起见, agent 没有依赖其他的 cloudmonitor-xxx 的基础镜像 手动定期维护 Dockerfile 的一致性

# 安装常用软件
RUN ln -snf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
  yum -q -y install epel-release && yum install -q -y sudo net-tools iproute dstat which supervisor stress unzip jq screen nginx wget telnet less && \
  yum -y clean all && \
  rm -rf /var/cache/yum

# install OSS util
RUN wget -qO /usr/local/bin/ossutil https://gosspublic.alicdn.com/ossutil/1.7.14/ossutil64

COPY scripts/docker/sc /usr/local/bin/
COPY scripts/docker/ensure_supervisord.sh /usr/local/bin/
COPY scripts/docker/supervisord.conf /etc/supervisord.conf
COPY gitcommit /usr/local/bin/holoinsight/bin/gitcommit

# 其他初始化
RUN echo 'export LANG=zh_CN.UTF-8' >> /etc/profile && \
  echo 'LC_ALL=zh_CN.UTF-8' >> /etc/profile && \
  echo 'PS1="\n\e[1;37m[\e[m\e[1;32m\u\e[m\e[1;33m@\e[m\e[1;35m\h\e[m \e[1;35m`hostname`\e[m \e[4m\`pwd\`\e[m\e[1;37m]\e[m\e[1;36m\e[m\n\\$ "' >> /etc/bashrc && \
  echo 'alias vim="vi"' >> /etc/bashrc && \
  echo 'alias ll="ls -laF"' >> /etc/bashrc && \
  echo 'shell /bin/bash' >> /root/.screenrc && \
  chmod a+x /usr/local/bin/ossutil

COPY scripts/docker/bin/app.ini /etc/supervisord.d/app.ini
COPY scripts/docker/bin/app.sh /usr/local/holoinsight/agent/bin/app.sh
COPY scripts/docker/bin/init_bashrc.sh /usr/local/holoinsight/agent/bin/init_bashrc.sh
COPY scripts/docker/bin/docker.sh /usr/local/bin/docker

COPY scripts/docker/entrypoint.sh /entrypoint.sh

COPY scripts/api /usr/local/holoinsight/agent/api
COPY scripts/docker/init.sh /usr/local/holoinsight/agent/bin/init.sh
COPY build/linux-amd64/bin/agent /usr/local/holoinsight/agent/bin/agent
COPY build/linux-amd64/bin/helper /usr/local/holoinsight/agent/bin/helper

VOLUME /usr/local/holoinsight/agent/data
VOLUME /usr/local/holoinsight/agent/logs

WORKDIR /usr/local/holoinsight/agent

RUN sh /usr/local/holoinsight/agent/bin/init.sh

ENTRYPOINT [ "/entrypoint.sh"]
