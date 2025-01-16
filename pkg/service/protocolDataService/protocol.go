package protocolDataService

import (
	"database/sql"
	"github.com/Layr-Labs/sidecar/internal/config"
	"github.com/Layr-Labs/sidecar/pkg/service/types"
	"github.com/Layr-Labs/sidecar/pkg/storage"
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

type StakerShares struct {
	Staker       string
	Strategy     string
	Shares       string
	BlockHeight  uint64
	Operator     *string
	Delegated    *bool
	AvsAddresses []string
}

// ListStakerShares returns the shares of a staker at a given block height, including the operator they were delegated to
// and the addresses of the AVSs the operator was registered to.
//
// If not blockHeight is provided, the most recently indexed block will be used.
func (pds *ProtocolDataService) ListStakerShares(staker string, blockHeight uint64) ([]*StakerShares, error) {
	shares := make([]*StakerShares, 0)

	if blockHeight == 0 {
		var currentBlock *storage.Block
		res := pds.db.Model(&storage.Block{}).Order("number desc").First(&currentBlock)
		if res.Error != nil {
			return nil, res.Error
		}
		blockHeight = currentBlock.Number
	}

	query := `
		with distinct_staker_strategies as (
			select
				ssd.staker,
				ssd.strategy,
				ssd.shares,
				ssd.block_number,
				row_number() over (partition by ssd.staker, ssd.strategy order by ssd.block_number desc) as rn
			from sidecar_mainnet_ethereum.staker_shares as ssd
			where
				ssd.staker = @staker
				and block_number <= @blockHeight
			order by block_number desc
		)
		select
			dss.*,
			dsc.operator,
			dsc.delegated,
			aosc.avs_list as avs_addresses
		from distinct_staker_strategies as dss
		left join lateral (
			select
				sdc.staker,
				sdc.operator,
				sdc.delegated,
				row_number() over (partition by sdc.staker order by sdc.block_number desc, sdc.log_index) as rn
			from sidecar_mainnet_ethereum.staker_delegation_changes as sdc
			where
				sdc.staker = dss.staker
				and sdc.block_number <= dss.block_number
			order by block_number desc
		) as dsc on (dsc.rn = 1)
		left join lateral (
			select
				jsonb_agg(distinct aosc.avs) as avs_list
			from sidecar_mainnet_ethereum.avs_operator_state_changes aosc
			where
				aosc.operator = dsc.operator
				and aosc.block_number <= dss.block_number
				and aosc.registered = true
		) as aosc on true
		where
			dss.rn = 1
		order by block_number desc;
	`
	res := pds.db.Raw(query,
		sql.Named("staker", staker),
		sql.Named("blockHeight", blockHeight),
	).Scan(&shares)
	if res.Error != nil {
		return nil, res.Error
	}
	return shares, nil
}
