#!/usr/bin/env bash

export GOTRACEBACK=all
export GODEBUG=gctrace=1,madvdontneed=1

exec /usr/local/holoinsight/agent/bin/agent
