package protocolDataService

import (
	"github.com/Layr-Labs/sidecar/internal/config"
	"github.com/Layr-Labs/sidecar/pkg/eigenState/stakerShares"
	"github.com/Layr-Labs/sidecar/pkg/service/types"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type ProtocolDataService struct {
	db           *gorm.DB
	logger       *zap.Logger
	globalConfig *config.Config
}

func NewProtocolDataService(
	db *gorm.DB,
	logger *zap.Logger,
	globalConfig *config.Config,
) *ProtocolDataService {
	return &ProtocolDataService{
		db:           db,
		logger:       logger,
		globalConfig: globalConfig,
	}
}

func (pds *ProtocolDataService) ListRegisteredAVSsForOperator(operator string, blockHeight uint64) (interface{}, error) {
	return nil, nil
}

func (pds *ProtocolDataService) ListDelegatedStrategiesForOperator(operator string, blockHeight uint64) (interface{}, error) {
	return nil, nil
}

func (pds *ProtocolDataService) GetOperatorDelegatedStake(operator string, strategy string, blockHeight uint64) (interface{}, error) {
	return nil, nil
}

func (pds *ProtocolDataService) ListDelegatedStakersForOperator(operator string, blockHeight uint64, pagination types.Pagination) (interface{}, error) {
	return nil, nil
}

func (pds *ProtocolDataService) ListStakerShares(staker string, blockHeight uint64) ([]*stakerShares.StakerShareDeltas, error) {
	shares := make([]*stakerShares.StakerShareDeltas, 0)

	whereParams := []interface{}{staker}
	where := "staker = ?"
	if blockHeight > 0 {
		where += " AND block_height <= ?"
		whereParams = append(whereParams, blockHeight)
	}

	res := pds.db.Model(&stakerShares.StakerShareDeltas{}).
		Where(where, whereParams...).
		Find(&shares)

	if res.Error != nil {
		return nil, res.Error
	}
	return shares, nil
}
