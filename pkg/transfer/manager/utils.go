/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package manager

import (
	"context"
	transferpb "github.com/traas-stack/holoinsight-agent/pkg/transfer/pb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"net"
	"time"
)

func MaybePrepareTransfer() {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	conn, err := grpc.DialContext(ctx, "", //
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(100*1024*1024)),
		grpc.WithTransportCredentials(insecure.NewCredentials()), //
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) { //
			return net.DialTimeout("unix", transferSockFile, 3*time.Second)
		}))

	if err != nil {
		return
	}
	defer conn.Close()

	transferpb.NewTransferSrviceClient(conn).Prepare(ctx, &transferpb.PingRequest{})
}
