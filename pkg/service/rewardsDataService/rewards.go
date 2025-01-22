package rewardsDataService

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/Layr-Labs/sidecar/internal/config"
	"github.com/Layr-Labs/sidecar/pkg/metaState/types"
	"github.com/Layr-Labs/sidecar/pkg/rewards"
	"github.com/Layr-Labs/sidecar/pkg/rewards/rewardsTypes"
	"github.com/Layr-Labs/sidecar/pkg/service/baseDataService"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type RewardsDataService struct {
	baseDataService.BaseDataService
	db                *gorm.DB
	logger            *zap.Logger
	globalConfig      *config.Config
	rewardsCalculator *rewards.RewardsCalculator
}

func NewRewardsDataService(
	db *gorm.DB,
	logger *zap.Logger,
	globalConfig *config.Config,
	rc *rewards.RewardsCalculator,
) *RewardsDataService {
	return &RewardsDataService{
		BaseDataService: baseDataService.BaseDataService{
			DB: db,
		},
		db:                db,
		logger:            logger,
		globalConfig:      globalConfig,
		rewardsCalculator: rc,
	}
}

func (rds *RewardsDataService) GetRewardsForSnapshot(ctx context.Context, snapshot string) ([]*rewardsTypes.Reward, error) {
	return rds.rewardsCalculator.FetchRewardsForSnapshot(snapshot)
}

type TotalClaimedReward struct {
	Earner string
	Token  string
	Amount string
}

func (rds *RewardsDataService) GetTotalClaimedRewards(ctx context.Context, earner string, blockHeight uint64) ([]*TotalClaimedReward, error) {
	blockHeight, err := rds.BaseDataService.GetCurrentBlockHeightIfNotPresent(ctx, blockHeight)
	if err != nil {
		return nil, err
	}

	query := `
		select
			earner,
			token,
			sum(claimed_amount) as amount
		from erwards_claimed as rc
		where
			earner = @earner
			and block_number <= @blockHeight
		group by 1, 2
	`

	claimedAmounts := make([]*TotalClaimedReward, 0)
	res := rds.db.Raw(query,
		sql.Named("earner", earner),
		sql.Named("block_height", blockHeight),
	).Scan(&claimedAmounts)

	if res.Error != nil {
		return nil, res.Error
	}
	return claimedAmounts, nil
}

// ListClaimedRewardsByBlockRange returns a list of claimed rewards for a given earner within a block range.
//
// If earner is an empty string, all claimed rewards within the block range are returned.
func (rds *RewardsDataService) ListClaimedRewardsByBlockRange(
	ctx context.Context,
	earner string,
	startBlockHeight uint64,
	endBlockHeight uint64,
) ([]*types.RewardsClaimed, error) {
	if endBlockHeight == 0 {
		return nil, fmt.Errorf("endBlockHeight must be greater than 0")
	}
	if endBlockHeight < startBlockHeight {
		return nil, fmt.Errorf("endBlockHeight must be greater than or equal to startBlockHeight")
	}

	query := `
		select
		    rc.root,
			rc.earner,
			rc.claimer,
			rc.recipient,
			rc.token,
			rc.claimed_amount,
			rc.transaction_hash,
			rc.block_number,
			rc.log_index
		from rewards_claimed as rc
		where
			block_number >= @startBlockHeight
			and block_number <= @endBlockHeight
	`
	args := []interface{}{
		sql.Named("startBlockHeight", startBlockHeight),
		sql.Named("endBlockHeight", endBlockHeight),
	}
	if earner != "" {
		query += " and earner = @earner"
		args = append(args, sql.Named("earner", earner))
	}
	query += " order by block_number, log_index"

	claimedRewards := make([]*types.RewardsClaimed, 0)
	res := rds.db.Raw(query, args...).Scan(&claimedRewards)

	if res.Error != nil {
		return nil, res.Error
	}
	return claimedRewards, nil
}
