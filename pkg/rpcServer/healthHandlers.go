package rpcServer

import (
	"context"
	sidecarV1 "github.com/Layr-Labs/protocol-apis/gen/protos/eigenlayer/sidecar/v1"
)

func (rpc *RpcServer) HealthCheck(ctx context.Context, req *sidecarV1.HealthCheckRequest) (*sidecarV1.HealthCheckResponse, error) {
	return &sidecarV1.HealthCheckResponse{
		Status: sidecarV1.HealthCheckResponse_SERVING,
	}, nil
}

func (rpc *RpcServer) ReadyCheck(ctx context.Context, req *sidecarV1.ReadyRequest) (*sidecarV1.ReadyResponse, error) {
	return &sidecarV1.ReadyResponse{
		Ready: true,
	}, nil
}
