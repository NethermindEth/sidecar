package rpcServer

import (
	"context"
	"errors"
	rewardsV1 "github.com/Layr-Labs/protocol-apis/gen/protos/eigenlayer/sidecar/v1/rewards"
	"github.com/Layr-Labs/sidecar/pkg/metaState/types"
	"github.com/Layr-Labs/sidecar/pkg/rewards"
	"github.com/Layr-Labs/sidecar/pkg/rewardsCalculatorQueue"
	"github.com/Layr-Labs/sidecar/pkg/service/rewardsDataService"
	"github.com/Layr-Labs/sidecar/pkg/utils"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (rpc *RpcServer) GetRewardsRoot(ctx context.Context, req *rewardsV1.GetRewardsRootRequest) (*rewardsV1.GetRewardsRootResponse, error) {
	return nil, status.Error(codes.Unimplemented, "method GetRewardsRoot not implemented")
}

func (rpc *RpcServer) GenerateRewards(ctx context.Context, req *rewardsV1.GenerateRewardsRequest) (*rewardsV1.GenerateRewardsResponse, error) {
	cutoffDate := req.GetCutoffDate()
	waitForComplete := req.GetWaitForComplete()

	var err error
	queued := false
	msg := rewardsCalculatorQueue.RewardsCalculationData{
		CalculationType: rewardsCalculatorQueue.RewardsCalculationType_CalculateRewards,
		CutoffDate:      cutoffDate,
	}

	if waitForComplete {
		data, qErr := rpc.rewardsQueue.EnqueueAndWait(ctx, msg)
		cutoffDate = data.CutoffDate
		err = qErr
	} else {
		rpc.rewardsQueue.Enqueue(&rewardsCalculatorQueue.RewardsCalculationMessage{
			Data:         msg,
			ResponseChan: make(chan *rewardsCalculatorQueue.RewardsCalculatorResponse),
		})
		queued = true
	}

	if err != nil {
		if errors.Is(err, &rewards.ErrRewardsCalculationInProgress{}) {
			return nil, status.Error(codes.FailedPrecondition, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &rewardsV1.GenerateRewardsResponse{
		CutoffDate: cutoffDate,
		Queued:     queued,
	}, nil
}

func (rpc *RpcServer) GenerateRewardsRoot(ctx context.Context, req *rewardsV1.GenerateRewardsRootRequest) (*rewardsV1.GenerateRewardsRootResponse, error) {
	cutoffDate := req.GetCutoffDate()
	if cutoffDate == "" {
		return nil, status.Error(codes.InvalidArgument, "snapshot date is required")
	}

	rpc.Logger.Sugar().Infow("Requesting rewards generation for snapshot date",
		zap.String("cutoffDate", cutoffDate),
	)
	_, err := rpc.rewardsQueue.EnqueueAndWait(context.Background(), rewardsCalculatorQueue.RewardsCalculationData{
		CalculationType: rewardsCalculatorQueue.RewardsCalculationType_CalculateRewards,
		CutoffDate:      cutoffDate,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	rpc.Logger.Sugar().Infow("Getting max snapshot for cutoff date",
		zap.String("cutoffDate", cutoffDate),
	)
	rewardsCalcEndDate, err := rpc.rewardsCalculator.GetMaxSnapshotDateForCutoffDate(cutoffDate)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if rewardsCalcEndDate == "" {
		return nil, status.Error(codes.NotFound, "no rewards calculated for the given snapshot date")
	}
	rpc.Logger.Sugar().Infow("Merkelizing rewards for snapshot date",
		zap.String("cutoffDate", cutoffDate),
		zap.String("rewardsCalcEndDate", rewardsCalcEndDate),
	)

	accountTree, _, _, err := rpc.rewardsCalculator.MerkelizeRewardsForSnapshot(rewardsCalcEndDate)
	if err != nil {
		rpc.Logger.Sugar().Errorw("failed to merkelize rewards for snapshot",
			zap.Error(err),
			zap.String("cutOffDate", cutoffDate),
			zap.String("rewardsCalcEndDate", rewardsCalcEndDate),
		)
		return nil, status.Error(codes.Internal, err.Error())
	}

	rootString := utils.ConvertBytesToString(accountTree.Root())
	rpc.Logger.Sugar().Infow("Rewards root generated",
		zap.String("root", rootString),
		zap.String("rewardsCalcEndDate", rewardsCalcEndDate),
		zap.String("cutoffDate", cutoffDate),
	)

	return &rewardsV1.GenerateRewardsRootResponse{
		RewardsRoot:        rootString,
		RewardsCalcEndDate: rewardsCalcEndDate,
	}, nil
}

func (rpc *RpcServer) GenerateStakerOperators(ctx context.Context, req *rewardsV1.GenerateStakerOperatorsRequest) (*rewardsV1.GenerateStakerOperatorsResponse, error) {
	cutoffDate := req.GetCutoffDate()

	if cutoffDate == "" {
		return nil, status.Error(codes.InvalidArgument, "snapshot date is required")
	}

	var err error
	queued := false
	msg := rewardsCalculatorQueue.RewardsCalculationData{
		CalculationType: rewardsCalculatorQueue.RewardsCalculationType_BackfillStakerOperatorsSnapshot,
		CutoffDate:      cutoffDate,
	}
	if req.GetWaitForComplete() {
		_, err = rpc.rewardsQueue.EnqueueAndWait(ctx, msg)
	} else {
		rpc.rewardsQueue.Enqueue(&rewardsCalculatorQueue.RewardsCalculationMessage{
			Data:         msg,
			ResponseChan: make(chan *rewardsCalculatorQueue.RewardsCalculatorResponse),
		})
		queued = true
	}

	if err != nil {
		if errors.Is(err, &rewards.ErrRewardsCalculationInProgress{}) {
			return nil, status.Error(codes.FailedPrecondition, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &rewardsV1.GenerateStakerOperatorsResponse{
		Queued: queued,
	}, nil
}

func (rpc *RpcServer) BackfillStakerOperators(ctx context.Context, req *rewardsV1.BackfillStakerOperatorsRequest) (*rewardsV1.BackfillStakerOperatorsResponse, error) {

	var err error
	queued := false
	msg := rewardsCalculatorQueue.RewardsCalculationData{
		CalculationType: rewardsCalculatorQueue.RewardsCalculationType_BackfillStakerOperators,
	}
	if req.GetWaitForComplete() {
		_, err = rpc.rewardsQueue.EnqueueAndWait(ctx, msg)
	} else {
		rpc.rewardsQueue.Enqueue(&rewardsCalculatorQueue.RewardsCalculationMessage{
			Data:         msg,
			ResponseChan: make(chan *rewardsCalculatorQueue.RewardsCalculatorResponse),
		})
		queued = true
	}

	if err != nil {
		if errors.Is(err, &rewards.ErrRewardsCalculationInProgress{}) {
			return nil, status.Error(codes.FailedPrecondition, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &rewardsV1.BackfillStakerOperatorsResponse{
		Queued: queued,
	}, nil
}

func (rpc *RpcServer) GetRewardsForSnapshot(ctx context.Context, req *rewardsV1.GetRewardsForSnapshotRequest) (*rewardsV1.GetRewardsForSnapshotResponse, error) {
	snapshot := req.GetSnapshot()
	if snapshot == "" {
		return nil, status.Error(codes.InvalidArgument, "snapshot is required")
	}

	snapshotRewards, err := rpc.rewardsDataService.GetRewardsForSnapshot(ctx, snapshot)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	rewardsRes := make([]*rewardsV1.Reward, 0, len(snapshotRewards))

	for _, reward := range snapshotRewards {
		rewardsRes = append(rewardsRes, &rewardsV1.Reward{
			Earner:   reward.Earner,
			Amount:   reward.CumulativeAmount,
			Snapshot: reward.Snapshot,
			Token:    reward.Token,
		})
	}

	return &rewardsV1.GetRewardsForSnapshotResponse{
		Rewards: rewardsRes,
	}, nil
}

func (rpc *RpcServer) GetAttributableRewardsForSnapshot(ctx context.Context, req *rewardsV1.GetAttributableRewardsForSnapshotRequest) (*rewardsV1.GetAttributableRewardsForSnapshotResponse, error) {
	return nil, status.Error(codes.Unimplemented, "method GetAttributableRewardsForSnapshot not implemented")
}

func (rpc *RpcServer) GetAttributableRewardsForDistributionRoot(ctx context.Context, req *rewardsV1.GetAttributableRewardsForDistributionRootRequest) (*rewardsV1.GetAttributableRewardsForDistributionRootResponse, error) {
	return nil, status.Error(codes.Unimplemented, "method GetAttributableRewardsForDistributionRoot not implemented")
}

func (rpc *RpcServer) GetAvailableRewards(ctx context.Context, req *rewardsV1.GetAvailableRewardsRequest) (*rewardsV1.GetAvailableRewardsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "method GetAvailableRewards not implemented")
}

// GetTotalClaimedRewards returns the total claimed rewards for an earner up to, and including, the provided block height.
func (rpc *RpcServer) GetTotalClaimedRewards(ctx context.Context, req *rewardsV1.GetTotalClaimedRewardsRequest) (*rewardsV1.GetTotalClaimedRewardsResponse, error) {
	earner := req.GetEarnerAddress()
	blockHeight := req.GetBlockHeight()

	if earner == "" {
		return nil, status.Error(codes.InvalidArgument, "earner address is required")
	}

	totalClaimed, err := rpc.rewardsDataService.GetTotalClaimedRewards(ctx, earner, blockHeight)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &rewardsV1.GetTotalClaimedRewardsResponse{
		Rewards: utils.Map(totalClaimed, func(r *rewardsDataService.TotalClaimedReward, i uint64) *rewardsV1.TotalClaimedReward {
			return &rewardsV1.TotalClaimedReward{
				Earner: r.Earner,
				Token:  r.Token,
				Amount: r.Amount,
			}
		}),
	}, nil
}

func (rpc *RpcServer) GetAvailableRewardsTokens(ctx context.Context, req *rewardsV1.GetAvailableRewardsTokensRequest) (*rewardsV1.GetAvailableRewardsTokensResponse, error) {
	return nil, status.Error(codes.Unimplemented, "method GetAvailableRewardsTokens not implemented")
}

func (rpc *RpcServer) GetSummarizedRewardsForEarner(ctx context.Context, req *rewardsV1.GetSummarizedRewardsForEarnerRequest) (*rewardsV1.GetSummarizedRewardsForEarnerResponse, error) {
	return nil, status.Error(codes.Unimplemented, "method GetSummarizedRewardsForEarner not implemented")
}

// GetClaimedRewardsByBlock returns the claimed rewards for an earner for a specific block.
func (rpc *RpcServer) GetClaimedRewardsByBlock(ctx context.Context, req *rewardsV1.GetClaimedRewardsByBlockRequest) (*rewardsV1.GetClaimedRewardsByBlockResponse, error) {
	blockHeight := req.GetBlockHeight()

	claims, err := rpc.rewardsDataService.ListClaimedRewardsByBlockRange(ctx, "", blockHeight, blockHeight)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &rewardsV1.GetClaimedRewardsByBlockResponse{
		Rewards: utils.Map(claims, func(c *types.RewardsClaimed, i uint64) *rewardsV1.ClaimedReward {
			return &rewardsV1.ClaimedReward{
				Earner:      c.Earner,
				Claimer:     c.Claimer,
				Token:       c.Token,
				BlockNumber: c.BlockNumber,
			}
		}),
	}, nil
}

// ListClaimedRewardsByBlockRange returns the claimed rewards for each block in the given range (inclusive of start and end block heights).
func (rpc *RpcServer) ListClaimedRewardsByBlockRange(ctx context.Context, req *rewardsV1.ListClaimedRewardsByBlockRangeRequest) (*rewardsV1.ListClaimedRewardsByBlockRangeResponse, error) {
	earner := req.GetEarnerAddress()
	startBlockHeight := req.GetStartBlockHeight()
	endBlockHeight := req.GetEndBlockHeight()

	if earner == "" {
		return nil, status.Error(codes.InvalidArgument, "earner address is required")
	}

	claims, err := rpc.rewardsDataService.ListClaimedRewardsByBlockRange(ctx, earner, startBlockHeight, endBlockHeight)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &rewardsV1.ListClaimedRewardsByBlockRangeResponse{
		Rewards: utils.Map(claims, func(c *types.RewardsClaimed, i uint64) *rewardsV1.ClaimedReward {
			return &rewardsV1.ClaimedReward{
				Earner:      c.Earner,
				Claimer:     c.Claimer,
				Token:       c.Token,
				BlockNumber: c.BlockNumber,
			}
		}),
	}, nil
}

func (rpc *RpcServer) ListDistributionRoots(ctx context.Context, req *rewardsV1.ListDistributionRootsRequest) (*rewardsV1.ListDistributionRootsResponse, error) {
	blockHeight := req.GetBlockHeight()
	roots, err := rpc.rewardsCalculator.ListDistributionRoots(blockHeight)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	responseRoots := make([]*rewardsV1.DistributionRoot, 0, len(roots))
	for _, root := range roots {
		responseRoots = append(responseRoots, &rewardsV1.DistributionRoot{
			Root:                      root.Root,
			RootIndex:                 root.RootIndex,
			RewardsCalculationEnd:     timestamppb.New(root.RewardsCalculationEnd),
			RewardsCalculationEndUnit: root.RewardsCalculationEndUnit,
			ActivatedAt:               timestamppb.New(root.ActivatedAt),
			ActivatedAtUnit:           root.ActivatedAtUnit,
			CreatedAtBlockNumber:      root.CreatedAtBlockNumber,
			TransactionHash:           root.TransactionHash,
			BlockHeight:               root.BlockNumber,
			LogIndex:                  root.LogIndex,
			Disabled:                  root.Disabled,
		})
	}

	return &rewardsV1.ListDistributionRootsResponse{
		DistributionRoots: responseRoots,
	}, nil
}
