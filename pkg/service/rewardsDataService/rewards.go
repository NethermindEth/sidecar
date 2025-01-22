package rewardsDataService

import (
	"context"
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
