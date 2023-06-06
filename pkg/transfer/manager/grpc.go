/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package manager

import (
	"context"
	transferpb "github.com/traas-stack/holoinsight-agent/pkg/transfer/pb"
)

type (
	transferSrviceServerImpl struct {
		transferpb.UnimplementedTransferSrviceServer
		tm *TransferManager
	}
)

func (t *transferSrviceServerImpl) Prepare(context.Context, *transferpb.PingRequest) (*transferpb.PingResponse, error) {
	t.tm.prepare()
	return &transferpb.PingResponse{}, nil
}

func (t *transferSrviceServerImpl) Ping(context.Context, *transferpb.PingRequest) (*transferpb.PingResponse, error) {
	return &transferpb.PingResponse{}, nil
}

func (t *transferSrviceServerImpl) StopAndSaveState(ctx context.Context, request *transferpb.StopAndSaveSaveRequest) (*transferpb.StopAndSaveSaveResponse, error) {
	state, err := t.tm.StopAndSaveState()
	return &transferpb.StopAndSaveSaveResponse{State: state}, err
}

func (t *transferSrviceServerImpl) TransferDone(ctx context.Context, request *transferpb.TransferDoneRequest) (*transferpb.TransferDoneResponse, error) {
	t.tm.TransferDone()
	return &transferpb.TransferDoneResponse{}, nil
}
