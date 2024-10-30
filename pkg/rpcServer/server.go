package rpcServer

import (
	"context"
	"github.com/Layr-Labs/go-sidecar/pkg/rewards"
	"github.com/Layr-Labs/go-sidecar/pkg/storage"

	"github.com/Layr-Labs/go-sidecar/pkg/eigenState/stateManager"
	v1 "github.com/Layr-Labs/protocol-apis/gen/protos/eigenlayer/sidecar/v1"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type RpcServer struct {
	v1.UnimplementedRpcServer
	v1.UnimplementedRewardsServer
	Logger            *zap.Logger
	blockStore        storage.BlockStore
	stateManager      *stateManager.EigenStateManager
	rewardsCalculator *rewards.RewardsCalculator
}

func NewRpcServer(
	ctx context.Context,
	grpcServer *grpc.Server,
	mux *runtime.ServeMux,
	bs storage.BlockStore,
	sm *stateManager.EigenStateManager,
	rc *rewards.RewardsCalculator,
	l *zap.Logger,
) (*RpcServer, error) {
	server := &RpcServer{
		blockStore:        bs,
		stateManager:      sm,
		rewardsCalculator: rc,
		Logger:            l,
	}

	v1.RegisterRpcServer(grpcServer, server)
	if err := v1.RegisterRpcHandlerServer(ctx, mux, server); err != nil {
		l.Sugar().Errorw("Failed to register SidecarRpc server", zap.Error(err))
		return nil, err
	}

	v1.RegisterRewardsServer(grpcServer, server)
	if err := v1.RegisterRewardsHandlerServer(ctx, mux, server); err != nil {
		l.Sugar().Errorw("Failed to register Rewards server", zap.Error(err))
		return nil, err
	}

	return server, nil
}
