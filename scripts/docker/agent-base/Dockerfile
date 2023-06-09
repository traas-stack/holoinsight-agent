FROM centos:7

# install common softwares
RUN yum -q -y install epel-release && yum install -q -y sudo net-tools iproute dstat which supervisor stress unzip jq screen wget telnet less && \
  yum -y clean all && \
  rm -rf /var/cache/yum

# setup bash
RUN echo 'PS1="\n\e[1;37m[\e[m\e[1;32m\u\e[m\e[1;33m@\e[m\e[1;35m\h\e[m \e[1;35m`hostname`\e[m \e[4m\`pwd\`\e[m\e[1;37m]\e[m\e[1;36m\e[m\n\\$ "' >> /etc/bashrc && \
  echo 'alias vim="vi"' >> /etc/bashrc && \
  echo 'alias ll="ls -laF"' >> /etc/bashrc && \
  echo 'shell /bin/bash' >> /root/.screenrc
