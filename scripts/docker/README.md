# 介绍
用于构建 agent docker 镜像.  
该脚本可以在 Mac m1 或 Linux 上执行, 要求机器上装有 docker. Mac m1 特别慢, 建议在 Linux 上执行, 可以通过 rsync 将代码复制到 Linux 上再构建.

```bash
./scripts/docker/build.sh your_agent_image:1.0.0
```

输出样例:
```text
[build agent bin using docker]
user home is /root
docker run --network host --platform=linux/amd64 --rm -v /root/workspace/remote/cloudmonitor-agent:/a -v /root/.cache/go-build:/root/.cache/go-build cloudmonitor-agent-build bash -c  cd /a && make agent helper
go build -ldflags "-s -w" -o build/linux-amd64/bin/agent -ldflags " -X cmd.goos=linux -X cmd.goarch=amd64" ./cmd/agent
go build -ldflags "-s -w" -o build/linux-amd64/bin/helper -ldflags " -X cmd.goos=linux -X cmd.goarch=amd64" ./cmd/containerhelper
[build agent docker image]
Sending build context to Docker daemon  172.7MB
Step 1/18 : FROM centos:7
 ---> eeb6ee3f44bd
Step 2/18 : RUN ln -snf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime &&   yum -q -y install epel-release && yum install -q -y sudo net-tools iproute dstat which supervisor stress unzip jq screen nginx wget telnet &&   yum -y clean all &&   rm -rf /var/cache/yum
 ---> Using cache
 ---> 9df3c3b523c0
Step 3/18 : COPY scripts/docker/sc /usr/local/bin/
 ---> Using cache
 ---> 302d75a2d4f3
Step 4/18 : COPY scripts/docker/ensure_supervisord.sh /usr/local/bin/
 ---> Using cache
 ---> 44a268d58928
Step 5/18 : COPY scripts/docker/supervisord.conf /etc/supervisord.conf
 ---> Using cache
 ---> 9f04b22b6dc0
Step 6/18 : RUN echo 'export LANG=zh_CN.UTF-8' >> /etc/profile &&   echo 'LC_ALL=zh_CN.UTF-8' >> /etc/profile &&   echo 'PS1="\n\e[1;37m[\e[m\e[1;32m\u\e[m\e[1;33m@\e[m\e[1;35m\h\e[m \e[1;35m`hostname`\e[m \e[4m\`pwd\`\e[m\e[1;37m]\e[m\e[1;36m\e[m\n\\$ "' >> /etc/bashrc &&   echo 'alias vim="vi"' >> /etc/bashrc &&   echo 'alias ll="ls -laF"' >> /etc/bashrc &&   echo 'shell /bin/bash' >> /root/.screenrc
 ---> Using cache
 ---> 55b38ae3ff64
Step 7/18 : COPY scripts/docker/bin/app.ini /etc/supervisord.d/app.ini
 ---> Using cache
 ---> b21f396bb6f8
Step 8/18 : COPY scripts/docker/bin/app.sh /usr/local/holoinsight/agent/bin/app.sh
 ---> Using cache
 ---> 94cb9927661e
Step 9/18 : COPY scripts/docker/entrypoint.sh /entrypoint.sh
 ---> Using cache
 ---> 2f1bf78bf17a
Step 10/18 : COPY scripts/api /usr/local/holoinsight/agent/api
 ---> Using cache
 ---> 2ad3dd0537d5
Step 11/18 : COPY scripts/docker/init.sh /usr/local/holoinsight/agent/bin/init.sh
 ---> Using cache
 ---> 4b919c36737d
Step 12/18 : COPY build/linux-amd64/bin/agent /usr/local/holoinsight/agent/bin/agent
 ---> Using cache
 ---> 8c4434ed6d01
Step 13/18 : COPY build/linux-amd64/bin/helper /usr/local/holoinsight/agent/bin/helper
 ---> Using cache
 ---> f36535cd094c
Step 14/18 : VOLUME /usr/local/holoinsight/agent/data
 ---> Using cache
 ---> 1a21d6686194
Step 15/18 : VOLUME /usr/local/holoinsight/agent/logs
 ---> Using cache
 ---> 92fd0d090311
Step 16/18 : WORKDIR /usr/local/holoinsight/agent
 ---> Using cache
 ---> 3a57276fcfd1
Step 17/18 : RUN sh /usr/local/holoinsight/agent/bin/init.sh
 ---> Using cache
 ---> 526bc4a6b8b1
Step 18/18 : ENTRYPOINT [ "/entrypoint.sh"]
 ---> Using cache
 ---> 7fcba39375d1
Successfully built 7fcba39375d1
Successfully tagged your_agent_image:1.0.0
```

构建完之后自己重新打tag, 然后上传到docker镜像仓库.  
