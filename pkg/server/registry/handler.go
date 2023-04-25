/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package registry

import (
	"context"
	"github.com/traas-stack/holoinsight-agent/pkg/server/registry/pb"
)

type (
	OnewayHandler    func(int32, []byte)
	RpcHandler       func(int32, []byte) (int32, interface{}, error)
	HandshakeHandler func(ctx context.Context, c pb.RegistryServiceForAgent_BiStreamsClient, version int64) error
	HandlerRegistry  struct {
		oneway    map[int32]OnewayHandler
		rpc       map[int32]RpcHandler
		handshake HandshakeHandler
	}
)

func NewHandlerRegistry() *HandlerRegistry {
	return &HandlerRegistry{
		oneway: make(map[int32]OnewayHandler),
		rpc:    make(map[int32]RpcHandler),
	}
}
func (r *HandlerRegistry) RegisterOneway(bizType int32, h OnewayHandler) {
	r.oneway[bizType] = h
}

func (r *HandlerRegistry) RegisterRpc(bizType int32, h RpcHandler) {
	r.rpc[bizType] = h
}

func (r *HandlerRegistry) RegisterHandshake(h HandshakeHandler) {
	r.handshake = h
}

func (r *HandlerRegistry) GetOneway(bizType int32) OnewayHandler {
	return r.oneway[bizType]
}

func (r *HandlerRegistry) GetRpc(bizType int32) RpcHandler {
	return r.rpc[bizType]
}

func (r *HandlerRegistry) GetHandshakeHandler() HandshakeHandler {
	return r.handshake
}
