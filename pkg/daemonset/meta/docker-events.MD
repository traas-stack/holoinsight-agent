# 介绍

# 几个比较重要的事件
1. create: 容器被创建, 一般之后跟随 start
2. start: 容器被启动
3. die: 容器由于各种原因退出 (主动或被动)
4. destroy: 容器销毁
5. oom: 容器尝试突破内存
6. kill/stop: 外部发起的kill, 比如 docker kill/stop


## 普通容器启动流程

## docker run

create
attach
start

如果有终端会收到
resize

容器主进程主动退出的 (exitcode任意都行)
die

## docker stop ...
stop 主动给容器发 kill, 然后容器die, 然后产生stop事件

会收到:
kill (可能会多次)
die
stop


docker rm 会收到
destroy


## exec

```bash
docker exec ... echo ok
```

会产生如下3个事件

```text
2022-11-02 21:47:07.317 [docker] [event] {"cid": "04ca5e8e16efccde3dc56966870513acde862bfdd0a91493ddfa8f390d7a6e6e", "action": "exec_create: echo ok"}
2022-11-02 21:47:07.317 [docker] [event] {"cid": "04ca5e8e16efccde3dc56966870513acde862bfdd0a91493ddfa8f390d7a6e6e", "action": "exec_start: echo ok"}
2022-11-02 21:47:07.378 [docker] [event] {"cid": "04ca5e8e16efccde3dc56966870513acde862bfdd0a91493ddfa8f390d7a6e6e", "action": "exec_die"}
```

## oom
当你在容器内, 尝试申请更多内存从而突破 memory limits 限制时, 会申请失败, 同时 docker 产生一个 oom 事件.

## docker restart
等价于 docker stop + docker start 的事件流, 最后再附加一个 restart 事件

kill
die
stop
start
restart

## docker cp to 容器
extract-to-dir

## docker cp from 容器
archive-path

# 参考资料
- https://docs.docker.com/engine/reference/commandline/events/
