# 介绍
这是一个通用的基于grpc的双向流.

1. 依赖 registry.Service
2. 不依赖业务实现, 业务实现是可以定制的
   1. 握手 handler
   2. 各种 rpc handler
