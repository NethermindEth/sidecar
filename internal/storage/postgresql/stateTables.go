package postgresql

import (
	"go.uber.org/zap"
	"golang.org/x/xerrors"
)

func (p *PostgresBlockStore) InsertIntoAvsOperatorChangesForBlock(blockNumber uint64) error {
	query := `
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
