#!/usr/bin/env bash
set -e

script_dir=$( dirname $0 )

SRC_DIR=$script_dir
DST_DIR=$script_dir

# TODO 如何控制产出的protobuf是v1的还是v2的?
protoc-3.14.0 \
  -I=$SRC_DIR \
  -I=$SRC_DIR/include \
  --go_out=paths=source_relative:$DST_DIR \
  --go-grpc_out=$DST_DIR \
  --go-grpc_opt=paths=source_relative \
  $SRC_DIR/*.proto
