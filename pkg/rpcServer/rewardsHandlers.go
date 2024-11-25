package rpcServer

import (
	"context"
	sidecarV1 "github.com/Layr-Labs/protocol-apis/gen/protos/eigenlayer/sidecar/v1"
	"github.com/Layr-Labs/sidecar/pkg/utils"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (rpc *RpcServer) GetRewardsRoot(ctx context.Context, req *sidecarV1.GetRewardsRootRequest) (*sidecarV1.GetRewardsRootResponse, error) {
	return nil, status.Error(codes.Unimplemented, "method GetRewardsRoot not implemented")
}

func (rpc *RpcServer) GenerateRewards(ctx context.Context, req *sidecarV1.GenerateRewardsRequest) (*sidecarV1.GenerateRewardsResponse, error) {
	cutoffDate := req.GetCutoffDate()

	if cutoffDate != "" {
		if err := rpc.rewardsCalculator.CalculateRewardsForSnapshotDate(cutoffDate); err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
	} else {
		sd, err := rpc.rewardsCalculator.CalculateRewardsForLatestSnapshot()
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
		cutoffDate = sd
	}
	return &sidecarV1.GenerateRewardsResponse{
		CutoffDate: cutoffDate,
	}, nil
}

func (rpc *RpcServer) GenerateRewardsRoot(ctx context.Context, req *sidecarV1.GenerateRewardsRootRequest) (*sidecarV1.GenerateRewardsRootResponse, error) {
	cutoffDate := req.GetCutoffDate()
	if cutoffDate == "" {
		return nil, status.Error(codes.InvalidArgument, "snapshot date is required")
	}

	err := rpc.rewardsCalculator.CalculateRewardsForSnapshotDate(cutoffDate)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	rewardsCalcEndDate, err := rpc.rewardsCalculator.GetMaxSnapshotDateForCutoffDate(cutoffDate)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if rewardsCalcEndDate == "" {
		return nil, status.Error(codes.NotFound, "no rewards calculated for the given snapshot date")
	}

	accountTree, _, err := rpc.rewardsCalculator.MerkelizeRewardsForSnapshot(rewardsCalcEndDate)
	if err != nil {
		rpc.Logger.Sugar().Errorw("failed to merkelize rewards for snapshot",
			zap.Error(err),
			zap.String("cutOffDate", cutoffDate),
			zap.String("rewardsCalcEndDate", rewardsCalcEndDate),
		)
		return nil, status.Error(codes.Internal, err.Error())
	}

	rootString := utils.ConvertBytesToString(accountTree.Root())

	return &sidecarV1.GenerateRewardsRootResponse{
		RewardsRoot:        rootString,
		RewardsCalcEndDate: rewardsCalcEndDate,
	}, nil
}

func (rpc *RpcServer) GetRewardsForSnapshot(ctx context.Context, req *sidecarV1.GetRewardsForSnapshotRequest) (*sidecarV1.GetRewardsForSnapshotResponse, error) {
	return nil, status.Error(codes.Unimplemented, "method GetRewardsForSnapshot not implemented")
}

func (rpc *RpcServer) GetAttributableRewardsForSnapshot(ctx context.Context, req *sidecarV1.GetAttributableRewardsForSnapshotRequest) (*sidecarV1.GetAttributableRewardsForSnapshotResponse, error) {
	return nil, status.Error(codes.Unimplemented, "method GetAttributableRewardsForSnapshot not implemented")
}

func (rpc *RpcServer) GetAttributableRewardsForDistributionRoot(ctx context.Context, req *sidecarV1.GetAttributableRewardsForDistributionRootRequest) (*sidecarV1.GetAttributableRewardsForDistributionRootResponse, error) {
	return nil, status.Error(codes.Unimplemented, "method GetAttributableRewardsForDistributionRoot not implemented")
}

func (rpc *RpcServer) GenerateClaimProof(ctx context.Context, req *sidecarV1.GenerateClaimProofRequest) (*sidecarV1.GenerateClaimProofResponse, error) {
	return nil, status.Error(codes.Unimplemented, "method GenerateClaimProof not implemented")
}

func (rpc *RpcServer) GetAvailableRewards(ctx context.Context, req *sidecarV1.GetAvailableRewardsRequest) (*sidecarV1.GetAvailableRewardsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "method GetAvailableRewards not implemented")
}

func (rpc *RpcServer) GetTotalClaimedRewards(ctx context.Context, req *sidecarV1.GetTotalClaimedRewardsRequest) (*sidecarV1.GetTotalClaimedRewardsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "method GetTotalClaimedRewards not implemented")
}

func (rpc *RpcServer) GetAvailableRewardsTokens(ctx context.Context, req *sidecarV1.GetAvailableRewardsTokensRequest) (*sidecarV1.GetAvailableRewardsTokensResponse, error) {
	return nil, status.Error(codes.Unimplemented, "method GetAvailableRewardsTokens not implemented")
}

func (rpc *RpcServer) GetSummarizedRewardsForEarner(ctx context.Context, req *sidecarV1.GetSummarizedRewardsForEarnerRequest) (*sidecarV1.GetSummarizedRewardsForEarnerResponse, error) {
	return nil, status.Error(codes.Unimplemented, "method GetSummarizedRewardsForEarner not implemented")
}

func (rpc *RpcServer) GetClaimedRewardsByBlock(ctx context.Context, req *sidecarV1.GetClaimedRewardsByBlockRequest) (*sidecarV1.GetClaimedRewardsByBlockResponse, error) {
	return nil, status.Error(codes.Unimplemented, "method GetClaimedRewardsByBlock not implemented")
}
