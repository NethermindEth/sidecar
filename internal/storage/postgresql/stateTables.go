package postgresql

import (
	"database/sql"
	"go.uber.org/zap"
	"golang.org/x/xerrors"
)

func (p *PostgresBlockStore) InsertIntoAvsOperatorChangesForBlock(blockNumber uint64) error {
	query := `
		with avs_operator_changes as (
			SELECT
				lower(t.arguments #>> '{0,Value}') as operator,
				lower(t.arguments #>> '{1,Value}') as avs,
				(case when (t.output_data -> 'status')::int = 1 then true else false) as registered,
				t.transaction_hash,
				t.log_index,
				t.block_number
			FROM transaction_logs t
			WHERE t.address = ?
			AND t.event_name = 'OperatorAVSRegistrationStatusUpdated'
			AND t.block_number = ?
		)
		select
			*
		into avs_operator_changes
		from avs_operator_changes
	`
	addressMap := p.GlobalConfig.GetContractsMapForEnvAndNetwork()
	if addressMap == nil {
		p.Logger.Sugar().Error("Failed to get contracts map for env and network")
		return xerrors.New("failed to get contracts map for env and network")
	}
	result := p.Db.Raw(query, addressMap.AvsDirectory, blockNumber)
	if result.Error != nil {
		p.Logger.Sugar().Errorw("Failed to insert into avs operator changes for block",
			zap.Error(result.Error),
			zap.Uint64("blockNumber", blockNumber),
		)
	}
	return nil
}

func (p *PostgresBlockStore) InsertIntoOperatorShareChangesForBlock(blockNumber uint64) error {
	query := `
		with operator_share_changes as (
			SELECT
				lower(t.arguments #>> '{0,Value}') as operator,
				lower(t.output_data ->> 'staker') as staker,
				lower(t.output_data ->> 'strategy') as strategy,
				case
					(when t.event_name = 'OperatorSharesIncreased' then (t.output_data ->> 'shares')::numeric(78,0) 
					else (t.output_data ->> 'shares')::numeric(78,0) * -1) as shares,
				t.transaction_hash,
				t.log_index,
				t.block_number
			FROM transaction_logs t
			WHERE t.address = ?
			AND t.event_name in('OperatorSharesIncreased', 'OperatorSharesDecreased')
			AND t.block_number = ?
		)
		select
		*
		into operator_share_changes
		from operator_share_changes
	`
	addressMap := p.GlobalConfig.GetContractsMapForEnvAndNetwork()
	if addressMap == nil {
		p.Logger.Sugar().Error("Failed to get contracts map for env and network")
		return xerrors.New("failed to get contracts map for env and network")
	}
	result := p.Db.Raw(query, addressMap.DelegationManager, blockNumber)
	if result.Error != nil {
		p.Logger.Sugar().Errorw("Failed to insert into operator share changes for block",
			zap.Error(result.Error),
			zap.Uint64("blockNumber", blockNumber),
		)
	}
	return nil
}

func (p *PostgresBlockStore) InsertIntoStakerShareChangesForBlock(blockNumber uint64) error {
	query := `
		with staker_deposits as (
			SELECT
				lower(coalesce(t.output_data ->> 'depositor', t.output_data ->> 'staker')) as staker,
				lower(t.output_data ->> 'strategy') as strategy,
				(t.output_data ->> 'shares')::numeric(78,0) as shares,
				t.transaction_hash,
				t.log_index,
				t.block_number
			FROM transaction_logs as t
			WHERE t.address = @strategyManagerAddress
			AND t.event_name 'Deposit'
			AND t.block_number = @blockNumber
		),
		staker_m1_withdrawals as (
			SELECT
				lower(coalesce(t.output_data ->> 'depositor', t.output_data ->> 'staker')) as staker,
				lower(t.output_data ->> 'strategy') as strategy,
				(t.output_data ->> 'shares')::numeric(78,0) * -1 as shares,
				t.transaction_hash,
				t.log_index,
				t.block_number
			FROM transaction_logs t
			WHERE t.address = @strategyManagerAddress
			AND t.event_name 'ShareWithdrawalQueued'
			AND t.block_number = @blockNumber
			-- Remove this transaction hash as it is the only withdrawal on m1 that was completed as shares. There is no corresponding deposit event. The withdrawal was completed to the same staker address.
			AND t.transaction_hash != '0x62eb0d0865b2636c74ed146e2d161e39e42b09bac7f86b8905fc7a830935dc1e'
		),
		staker_m2_withdrawals as (
			WITH migrations AS (
			  SELECT 
				(
				  SELECT lower(string_agg(lpad(to_hex(elem::int), 2, '0'), ''))
				  FROM jsonb_array_elements_text(t.output_data->'oldWithdrawalRoot') AS elem
				) AS m1_withdrawal_root,
				(
				  SELECT lower(string_agg(lpad(to_hex(elem::int), 2, '0'), ''))
				  FROM jsonb_array_elements_text(t.output_data->'newWithdrawalRoot') AS elem
				) AS m2_withdrawal_root
			  FROM transaction_logs t
			  WHERE t.address = @delegationManagerAddress
			  AND t.event_name = 'WithdrawalMigrated'
			),
			full_m2_withdrawals AS (
			  SELECT
				lower(t.output_data #>> '{withdrawal}') as withdrawals,
				(
				  SELECT lower(string_agg(lpad(to_hex(elem::int), 2, '0'), ''))
				  FROM jsonb_array_elements_text(t.output_data ->'withdrawalRoot') AS elem
				) AS withdrawal_root,
				lower(t.output_data #>> '{withdrawal, staker}') AS staker,
				lower(t_strategy.strategy) AS strategy,
				(t_share.share)::numeric(78,0) AS shares,
				t_strategy.strategy_index,
				t_share.share_index,
				t.transaction_hash,
				t.log_index,
				t.block_number
			  FROM transaction_logs t
			  left join blocks as b on (t.block_sequence_id = b.id),
			  	jsonb_array_elements_text(t.output_data #> '{withdrawal, strategies}') WITH ORDINALITY AS t_strategy(strategy, strategy_index),
			  	jsonb_array_elements_text(t.output_data #> '{withdrawal, shares}') WITH ORDINALITY AS t_share(share, share_index)
			  WHERE t.address = @delegationManagerAddress
			  AND t.event_name = 'WithdrawalQueued'
			  AND t_strategy.strategy_index = t_share.share_index
			  AND t.block_number = @blockNumber
			)
			-- Parse out the m2 withdrawals that were migrated from m1
			SELECT 
			  full_m2_withdrawals.*
			FROM 
			  full_m2_withdrawals
			LEFT JOIN 
			  migrations 
			ON 
			  full_m2_withdrawals.withdrawal_root = migrations.m2_withdrawal_root
			WHERE 
			  migrations.m2_withdrawal_root IS NULL
		),
		eigenpod_shares as (
			SELECT
				lower(t.arguments #>> '{0,Value}') AS staker,
				(t.output_data ->> 'sharesDelta')::numeric(78,0) as shares,
				t.transaction_hash,
				t.log_index,
				t.block_number
			FROM transaction_logs t
			WHERE t.address = @eigenpodManagerAddress
			AND t.event_name = 'PodSharesUpdated'
			AND t.block_number = @blockNumber
		)
		combined_results as (
			SELECT staker, strategy, shares, 0 as strategy_index, transaction_hash, log_index, block_number
			FROM staker_deposits
		
			UNION ALL
		
			-- Subtract m1 & m2 withdrawals
			SELECT staker, strategy, shares * -1, 0 as strategy_index, transaction_hash, log_index, block_number
			FROM staker_m1_withdrawals
		
			UNION ALL
		
			SELECT staker, strategy, shares * -1, strategy_index, transaction_hash, log_index, block_number
			FROM staker_m2_withdrawals
		
			UNION all
		
			-- Shares in eigenpod are positive or negative, so no need to multiply by -1
			SELECT staker, '0xbeac0eeeeeeeeeeeeeeeeeeeeeeeeeeeeeebeac0' as strategy, shares, 0 as strategy_index, transaction_hash, log_index, block_number
			FROM eigenpod_shares
		)
		select
		*
		into staker_share_changes
		from combined_results
	`
	addressMap := p.GlobalConfig.GetContractsMapForEnvAndNetwork()
	if addressMap == nil {
		p.Logger.Sugar().Error("Failed to get contracts map for env and network")
		return xerrors.New("failed to get contracts map for env and network")
	}
	result := p.Db.Raw(query,
		sql.Named("strategyManagerAddress", addressMap.StrategyManager),
		sql.Named("delegationManagerAddress", addressMap.DelegationManager),
		sql.Named("eigenpodManagerAddress", addressMap.EigenpodManager),
		sql.Named("blockNumber", blockNumber),
	)
	if result.Error != nil {
		p.Logger.Sugar().Errorw("Failed to insert into staker share changes for block",
			zap.Error(result.Error),
			zap.Uint64("blockNumber", blockNumber),
		)
	}
	return nil
}

func (p *PostgresBlockStore) InsertIntoStakerDelegationChangesForBlock(blockNumber uint64) error {
	query := `
		with staker_delegations as (
			SELECT
				lower(t.arguments #>> '{0,Value}') as staker,
				lower(t.output_data ->> 'strategy') as strategy,
				case when t.event_name = 'StakerDelegated' then true else false end as delegated,
				t.transaction_hash,
				t.log_index,
				t.block_number
			FROM transaction_logs t
			WHERE t.address = ?
			AND t.event_name in('StakerDelegated', 'StakerUndelegated')
			AND t.block_number = ?
		)
		select
		*
		into staker_delegation_changes
		from staker_delegations
	`
	addressMap := p.GlobalConfig.GetContractsMapForEnvAndNetwork()
	if addressMap == nil {
		p.Logger.Sugar().Error("Failed to get contracts map for env and network")
		return xerrors.New("failed to get contracts map for env and network")
	}
	result := p.Db.Raw(query, addressMap.DelegationManager, blockNumber)
	if result.Error != nil {
		p.Logger.Sugar().Errorw("Failed to insert into staker share changes for block",
			zap.Error(result.Error),
			zap.Uint64("blockNumber", blockNumber),
		)
	}
	return nil
}

func (p *PostgresBlockStore) InsertIntoActiveRewardSubmissionsForBlock(blockNumber uint64) error {
	query := `
		with rows_to_insert as (
			SELECT
				lower(tl.arguments #>> '{0,Value}') AS avs,
				lower(tl.arguments #>> '{2,Value}') AS reward_hash,
				coalesce(lower(tl.output_data #>> '{rewardsSubmission}'), lower(tl.output_data #>> '{rangePayment}')) as rewards_submission,
				coalesce(lower(tl.output_data #>> '{rewardsSubmission, token}'), lower(tl.output_data #>> '{rangePayment, token}')) as token,
				coalesce(tl.output_data #>> '{rewardsSubmission,amount}', tl.output_data #>> '{rangePayment,amount}')::numeric(78,0) as amount,
				to_timestamp(coalesce(tl.output_data #>> '{rewardsSubmission,startTimestamp}', tl.output_data #>> '{rangePayment,startTimestamp}')::bigint)::timestamp(6) as start_timestamp,
				coalesce(tl.output_data #>> '{rewardsSubmission,duration}', tl.output_data #>> '{rangePayment,duration}')::bigint as duration,
				to_timestamp(
						coalesce(tl.output_data #>> '{rewardsSubmission,startTimestamp}', tl.output_data #>> '{rangePayment,startTimestamp}')::bigint
							+ coalesce(tl.output_data #>> '{rewardsSubmission,duration}', tl.output_data #>> '{rangePayment,duration}')::bigint
				)::timestamp(6) as end_timestamp,
				lower(t.entry ->> 'strategy') as strategy,
				(t.entry ->> 'multiplier')::numeric(78,0) as multiplier,
				t.strategy_index as strategy_index,
				tl.transaction_hash,
				tl.log_index,
				tl.block_number
			FROM transaction_logs tl
				CROSS JOIN LATERAL jsonb_array_elements(
					coalesce(tl.output_data #> '{rewardsSubmission,strategiesAndMultipliers}',tl.output_data #> '{rangePayment,strategiesAndMultipliers}')
				) WITH ORDINALITY AS t(entry, strategy_index)
			WHERE address = ?
			AND (event_name = 'AVSRewardsSubmissionCreated' or event_name = 'RangePaymentCreated')
			AND block_number = ?
		)
		select
		*
		into active_reward_submissions
		from rows_to_insert
	`
	addressMap := p.GlobalConfig.GetContractsMapForEnvAndNetwork()
	if addressMap == nil {
		p.Logger.Sugar().Error("Failed to get contracts map for env and network")
		return xerrors.New("failed to get contracts map for env and network")
	}
	result := p.Db.Raw(query, addressMap.RewardsCoordinator, blockNumber)
	if result.Error != nil {
		p.Logger.Sugar().Errorw("Failed to insert into avs operator changes for block",
			zap.Error(result.Error),
			zap.Uint64("blockNumber", blockNumber),
		)
	}
	return nil
}
