# Docker events

## Important events

- start: container started
- oom: container hits the memory limit
- kill: send signal to container. If it is a SIGTERM/SIGKILL, we will receive die event soon
- die: container exited for various reasons
- stop: execute docker stop explicitly
- destroy: `docker rm`

## Common operations and generated events

The following discussion omits some unimportant events.

### docker run

generate events

- create
- attach
- start

If there is a tty:

- resize

If container exit:

- die

If run with '--rm' option:

- destroy

Otherwise, you need to explicitly docker rm:

- destroy

### docker stop ...

- kill
- die
- stop

Executing `docker stop` has no effect on exited containers

### docker rm

Execute `docker rm` when the container has exited

- destroy

Execute `docker rm -f` when the container is running normally

- kill
- die
- destroy

### docker kill

Executing `docker kill` when the container is running normally

- kill

If the container dies due to this it will immediately receive

- die

Executing `docker kill` for exited containers will fail.

### docker exec

```bash
docker exec ... echo ok
```

The following 3 events will be generated

```text
2022-11-02 21:47:07.317 [docker] [event] {"cid": "04ca5e8e16efccde3dc56966870513acde862bfdd0a91493ddfa8f390d7a6e6e", "action": "exec_create: echo ok"}
2022-11-02 21:47:07.317 [docker] [event] {"cid": "04ca5e8e16efccde3dc56966870513acde862bfdd0a91493ddfa8f390d7a6e6e", "action": "exec_start: echo ok"}
2022-11-02 21:47:07.378 [docker] [event] {"cid": "04ca5e8e16efccde3dc56966870513acde862bfdd0a91493ddfa8f390d7a6e6e", "action": "exec_die"}
```

## docker restart

docker restart = docker stop + docker start + restart event

- kill
- die
- stop
- start
- restart

### docker cp to container

- extract-to-dir

### docker cp from container

- archive-path

# References

- https://docs.docker.com/engine/reference/commandline/events/
