package rewardsDataService

import (
	"context"
	"database/sql"
	"github.com/Layr-Labs/sidecar/internal/config"
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
