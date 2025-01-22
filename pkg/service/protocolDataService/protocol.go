package protocolDataService

import (
	"context"
	"database/sql"
	"github.com/Layr-Labs/sidecar/internal/config"
	"github.com/Layr-Labs/sidecar/pkg/service/baseDataService"
	"github.com/Layr-Labs/sidecar/pkg/service/types"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"strings"
	"sync"
)

type ProtocolDataService struct {
	baseDataService.BaseDataService
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
		BaseDataService: baseDataService.BaseDataService{
			DB: db,
		},
		db:           db,
		logger:       logger,
		globalConfig: globalConfig,
	}
}

func (pds *ProtocolDataService) ListRegisteredAVSsForOperator(ctx context.Context, operator string, blockHeight uint64) ([]string, error) {
	operator = strings.ToLower(operator)

	blockHeight, err := pds.BaseDataService.GetCurrentBlockHeightIfNotPresent(blockHeight)
	if err != nil {
		return nil, err
	}

	query := `
		with ranked_operators as (
			select
				aosc.operator,
				aosc.avs,
				aosc.registered,
				row_number() over (partition by aosc.operator order by aosc.block_number desc, aosc.log_index asc) as rn
			from avs_operator_state_changes as aosc
			where
				operator = @operator'
				and block_number <= @blockHeight
		)
		select
			distinct ro.avs as avs
		from ranked_operators as ro
		where
			ro.rn = 1
			and ro.registered = true
	`
	var avsAddresses []string
	res := pds.db.Raw(query,
		sql.Named("operator", operator),
		sql.Named("blockHeight", blockHeight),
	).Scan(&avsAddresses)

	if res.Error != nil {
		return nil, res.Error
	}
	return avsAddresses, nil
}

func (pds *ProtocolDataService) ListDelegatedStrategiesForOperator(ctx context.Context, operator string, blockHeight uint64) ([]string, error) {
	operator = strings.ToLower(operator)
	blockHeight, err := pds.BaseDataService.GetCurrentBlockHeightIfNotPresent(blockHeight)
	if err != nil {
		return nil, err
	}

	query := `
		with operator_stakers as (
			select distinct on (staker)
				staker,
				block_number,
				delegated
			from staker_delegation_changes
			where
				operator = @operator
				and block_number <= @blockHeight
			order by staker, block_number desc, log_index asc
		),
		delegated_stakers as (
			select
				staker,
				block_number
			from operator_stakers
			where delegated = true
		),
		staker_strategies as (
			select
				s.strategy,
				s.shares
			from delegated_stakers as ds
			left join staker_share_deltas as s
				on s.staker = ds.staker
				and s.block_number <= ds.block_number
		),
		strategy_shares as (
			select
				ss.strategy,
				sum(ss.shares) as shares
			from staker_strategies as ss
			group by 1
		)
		select
			strategy
		from strategy_shares
		where shares > 0;
	`

	var strategies []string
	res := pds.db.Raw(query,
		sql.Named("operator", operator),
		sql.Named("blockHeight", blockHeight),
	).Scan(&strategies)

	if res.Error != nil {
		return nil, res.Error
	}
	return strategies, nil
}

// getTotalDelegatedOperatorSharesForStrategy returns the total shares delegated to an operator for a given strategy at a given block height.
func (pds *ProtocolDataService) getTotalDelegatedOperatorSharesForStrategy(ctx context.Context, operator string, strategy string, blockHeight uint64) (string, error) {
	query := `
		with operator_stakers as (
			select
				staker,
				operator,
				delegated,
				block_number,
				log_index,
				row_number() over (partition by staker order by block_number desc, log_index asc) as rn
			from staker_delegation_changes
			where
				operator = @operator
				and block_number <= @blockHeight
			order by block_number desc, log_index desc
		),
		distinct_delegated_stakers as (
			select
				distinct staker,
				operator,
				block_number,
				log_index
			from operator_stakers as os
			where
				os.rn = 1
				and os.delegated = true
		),
		stakers_with_shares as (
			select
				dds.staker,
				dds.operator,
				dds.block_number,
				ss.strategy,
				dds.log_index,
				ss.shares
			from distinct_delegated_stakers as dds
			join lateral (
				select
					ssd.strategy,
					sum(ssd.shares) as shares
				-- TODO: this should reference the staker_shares table once it is persistent and not deleted and recreated on each rewards run
				from staker_share_deltas as ssd
				where
					ssd.staker = dds.staker
					and ssd.block_number <= dds.block_number
				group by 1
			) as ss on(ss.strategy = @strategy)
		)
		select
			sws.operator,
			sum(sws.shares) as shares
		from stakers_with_shares as sws
		group by 1
	`

	var results struct {
		Operator string
		Shares   string
	}

	res := pds.db.Raw(query,
		sql.Named("operator", strings.ToLower(operator)),
		sql.Named("strategy", strings.ToLower(strategy)),
		sql.Named("blockHeight", blockHeight),
	).Scan(&results)

	if res.Error != nil {
		return "", res.Error
	}
	return results.Shares, nil
}

type OperatorDelegatedStake struct {
	Shares       string
	AvsAddresses []string
}

type ResultCollector[T any] struct {
	Result T
	Error  error
}

func (pds *ProtocolDataService) GetOperatorDelegatedStake(ctx context.Context, operator string, strategy string, blockHeight uint64) (*OperatorDelegatedStake, error) {
	blockHeight, err := pds.BaseDataService.GetCurrentBlockHeightIfNotPresent(blockHeight)
	if err != nil {
		return nil, err
	}

	var wg sync.WaitGroup
	sharesChan := make(chan *ResultCollector[string])
	avsChan := make(chan *ResultCollector[[]string])

	wg.Add(2)

	go func() {
		defer wg.Done()
		result := &ResultCollector[string]{}

		shares, err := pds.getTotalDelegatedOperatorSharesForStrategy(ctx, operator, strategy, blockHeight)
		if err != nil {
			result.Error = err
		} else {
			result.Result = shares
		}
		sharesChan <- result
	}()

	go func() {
		defer wg.Done()
		result := &ResultCollector[[]string]{}

		avsAddresses, err := pds.ListRegisteredAVSsForOperator(ctx, operator, blockHeight)
		if err != nil {
			result.Error = err
		} else {
			result.Result = avsAddresses
		}
		avsChan <- result
	}()
	wg.Wait()
	close(sharesChan)
	close(avsChan)

	shares := <-sharesChan
	if shares.Error != nil {
		pds.logger.Sugar().Errorw("Failed to get operator delegated stake",
			zap.String("operator", operator),
			zap.String("strategy", strategy),
			zap.Uint64("blockHeight", blockHeight),
			zap.Error(shares.Error),
		)
		return nil, shares.Error
	}

	registeredAvss := <-avsChan
	if registeredAvss.Error != nil {
		pds.logger.Sugar().Errorw("Failed to get registered AVSs for operator",
			zap.String("operator", operator),
			zap.String("strategy", strategy),
			zap.Uint64("blockHeight", blockHeight),
			zap.Error(registeredAvss.Error),
		)
		return nil, registeredAvss.Error
	}

	return &OperatorDelegatedStake{
		Shares:       shares.Result,
		AvsAddresses: registeredAvss.Result,
	}, nil
}

func (pds *ProtocolDataService) ListDelegatedStakersForOperator(ctx context.Context, operator string, blockHeight uint64, pagination *types.Pagination) ([]string, error) {
	bh, err := pds.BaseDataService.GetCurrentBlockHeightIfNotPresent(blockHeight)
	if err != nil {
		return nil, err
	}

	query := `
		with staker_operator_delegations as (
			SELECT DISTINCT ON (staker)
				staker,
				operator,
				delegated
			FROM staker_delegation_changes
			WHERE operator = @operator
				AND block_number <= @blockHeight
			ORDER BY staker, block_number desc, log_index asc
		)
		SELECT
			sod.staker
		from staker_operator_delegations as sod
		where sod.delegated = true
	`

	queryParams := []interface{}{
		sql.Named("operator", operator),
		sql.Named("blockHeight", bh),
	}

	if pagination != nil {
		query += ` LIMIT @limit`
		queryParams = append(queryParams, sql.Named("limit", pagination.PageSize))

		if pagination.Page > 0 {
			query += ` OFFSET @offset`
			queryParams = append(queryParams, sql.Named("offset", pagination.Page*pagination.PageSize))
		}
	}

	var stakers []string
	res := pds.db.Raw(query, queryParams...).Scan(&stakers)
	if res.Error != nil {
		return nil, res.Error
	}
	return stakers, nil
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
func (pds *ProtocolDataService) ListStakerShares(ctx context.Context, staker string, blockHeight uint64) ([]*StakerShares, error) {
	shares := make([]*StakerShares, 0)

	bh, err := pds.BaseDataService.GetCurrentBlockHeightIfNotPresent(blockHeight)
	if err != nil {
		return nil, err
	}

	query := `
		with distinct_staker_strategies as (
			select
				ssd.staker,
				ssd.strategy,
				ssd.shares,
				ssd.block_number,
				row_number() over (partition by ssd.staker, ssd.strategy order by ssd.block_number desc) as rn
			from staker_shares as ssd
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
			from staker_delegation_changes as sdc
			where
				sdc.staker = dss.staker
				and sdc.block_number <= dss.block_number
			order by block_number desc
		) as dsc on (dsc.rn = 1)
		left join lateral (
			select
				jsonb_agg(distinct aosc.avs) as avs_list
			from avs_operator_state_changes aosc
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
		sql.Named("blockHeight", bh),
	).Scan(&shares)
	if res.Error != nil {
		return nil, res.Error
	}
	return shares, nil
}
