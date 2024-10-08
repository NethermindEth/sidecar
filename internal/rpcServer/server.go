package rpcServer

import (
	"context"

	"github.com/Layr-Labs/go-sidecar/internal/eigenState/stateManager"
	"github.com/Layr-Labs/go-sidecar/internal/storage"
	v1 "github.com/Layr-Labs/sidecar-apis/protos/eigenlayer/sidecar/v1"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type RpcServer struct {
	v1.UnimplementedRpcServer
	Logger       *zap.Logger
	blockStore   storage.BlockStore
	stateManager *stateManager.EigenStateManager
}

func NewRpcServer(
	ctx context.Context,
	grpcServer *grpc.Server,
	mux *runtime.ServeMux,
	bs storage.BlockStore,
	sm *stateManager.EigenStateManager,
	l *zap.Logger,
) (*RpcServer, error) {
	server := &RpcServer{
		blockStore:   bs,
		stateManager: sm,
		Logger:       l,
	}

	v1.RegisterRpcServer(grpcServer, server)

	if err := v1.RegisterRpcHandlerServer(ctx, mux, server); err != nil {
		l.Sugar().Errorw("Failed to register SidecarRpc server", zap.Error(err))
		return nil, err
	}

	return server, nil
}
