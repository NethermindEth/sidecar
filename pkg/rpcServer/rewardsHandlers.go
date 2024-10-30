package rpcServer

import (
	"context"
	sidecarV1 "github.com/Layr-Labs/protocol-apis/gen/protos/eigenlayer/sidecar/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (rpc *RpcServer) GenerateRewards(ctx context.Context, req *sidecarV1.GenerateRewardsRequest) (*sidecarV1.GenerateRewardsResponse, error) {
	snapshotDate := req.GetSnapshot()

	if snapshotDate != "" {
		if err := rpc.rewardsCalculator.CalculateRewardsForSnapshotDate(snapshotDate); err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
	} else {
		sd, err := rpc.rewardsCalculator.CalculateRewardsForLatestSnapshot()
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
		snapshotDate = sd
	}
	return &sidecarV1.GenerateRewardsResponse{
		Snapshot: snapshotDate,
	}, nil
}
