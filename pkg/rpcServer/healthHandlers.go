package rpcServer

import (
	"context"
	healthV1 "github.com/Layr-Labs/protocol-apis/gen/protos/eigenlayer/sidecar/v1/health"
)

func (rpc *RpcServer) HealthCheck(ctx context.Context, req *healthV1.HealthCheckRequest) (*healthV1.HealthCheckResponse, error) {
	return &healthV1.HealthCheckResponse{
		Status: healthV1.HealthCheckResponse_SERVING,
	}, nil
}

func (rpc *RpcServer) ReadyCheck(ctx context.Context, req *healthV1.ReadyRequest) (*healthV1.ReadyResponse, error) {
	return &healthV1.ReadyResponse{
		Ready: true,
	}, nil
}
