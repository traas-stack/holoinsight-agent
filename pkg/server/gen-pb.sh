#!/usr/bin/env bash
set -e

# Doc: We need to generate protobuf golang files from `*.proto`.
# Originally, this required various binaries(protoc/protoc-gen-go/protoc-gen-java...) to be installed on the machine where the command was executed.
# This is too complex and prone to version inconsistencies.

# https://github.com/namely/docker-protoc provides a docker image containing various binaries required to generate protobuf.

cd `dirname $0`

cid=`docker run -d --rm --entrypoint=sleep -v $PWD:/defs namely/protoc-all 3600`

docker exec -it $cid /usr/local/bin/entrypoint.sh -l go -o . --go-source-relative -f pb/common.proto
docker exec -it $cid /usr/local/bin/entrypoint.sh -i /defs/pb -l go -o . --go-source-relative -f gateway/pb/gateway-for-agent.proto
docker exec -it $cid /usr/local/bin/entrypoint.sh -i /defs/pb -l go -o . --go-source-relative -f registry/pb/registry-for-agent.proto
docker exec -it $cid /usr/local/bin/entrypoint.sh -i /defs/pb -l go -o . --go-source-relative -f registry/pb/registry-for-prod.proto

docker stop $cid
echo done
