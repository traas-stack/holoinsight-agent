FROM holoinsight/agent-builder:1.0.1 as builder
ARG GO111MODULE=on
ARG GOPROXY=direct
# Users in China can use GOPROXY to speed up building.
# ARG GOPROXY=https://goproxy.cn,direct
RUN env
COPY . /workspace

RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/root/go/pkg \
    /workspace/scripts/build/build-using-go.sh && \
  mv /workspace/build/`go env GOOS`-`go env GOARCH`/bin/agent /agent && \
  mv /workspace/build/`go env GOOS`-`go env GOARCH`/bin/helper /helper

FROM holoinsight/agent-base:1.0.0

COPY scripts/docker/sc /usr/local/bin/
COPY scripts/docker/ensure_supervisord.sh /usr/local/bin/
COPY scripts/docker/supervisord.conf /etc/supervisord.conf
COPY gitcommit /usr/local/bin/holoinsight/bin/gitcommit

COPY scripts/docker/bin/app.ini /etc/supervisord.d/app.ini
COPY scripts/docker/bin/app.sh /usr/local/holoinsight/agent/bin/app.sh
COPY scripts/docker/bin/init_bashrc.sh /usr/local/holoinsight/agent/bin/init_bashrc.sh
COPY scripts/docker/bin/docker.sh /usr/local/bin/docker

COPY scripts/docker/entrypoint.sh /entrypoint.sh

COPY scripts/api /usr/local/holoinsight/agent/api
COPY scripts/docker/init.sh /usr/local/holoinsight/agent/bin/init.sh
COPY --from=builder /agent /usr/local/holoinsight/agent/bin/agent
COPY --from=builder /helper /usr/local/holoinsight/agent/bin/helper

VOLUME /usr/local/holoinsight/agent/data
VOLUME /usr/local/holoinsight/agent/logs

WORKDIR /usr/local/holoinsight/agent

RUN sh /usr/local/holoinsight/agent/bin/init.sh

ENTRYPOINT [ "/entrypoint.sh"]
