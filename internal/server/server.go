package server

import (
	"context"
	"github.com/Layr-Labs/sidecar/internal/backfiller"
	v1 "github.com/Layr-Labs/sidecar/protos/eigenlayer/blocklake/v1"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type BackfillServer struct {
	v1.UnimplementedBackfillerServer
	Logger     *zap.Logger
	Backfiller *backfiller.Backfiller
}

func NewBackfillServer(
	ctx context.Context,
	grpcServer *grpc.Server,
	mux *runtime.ServeMux,
	l *zap.Logger,
	b *backfiller.Backfiller,
) (*BackfillServer, error) {
	bfServer := &BackfillServer{
		Logger:     l,
		Backfiller: b,
	}

	v1.RegisterBackfillerServer(grpcServer, bfServer)

	err := v1.RegisterBackfillerHandlerServer(ctx, mux, bfServer)
	if err != nil {
		l.Error("Failed to register gateway", zap.Error(err))
		return nil, err
	}

	return bfServer, nil
}
