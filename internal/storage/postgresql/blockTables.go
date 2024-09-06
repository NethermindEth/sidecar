package postgresql

import "database/sql"

func (p *PostgresBlockStore) CloneRegisteredAvsOperatorsForNewBlock(newBlockNumber uint64) error {
	query := `
		insert into registered_avs_operators (avs, operator, block_number)
			select
				operator,
				avs,
				@newBlockNumber as block_number
			from registered_avs_operators where block_number = @previousBlockNumber
	`
	results := p.Db.Exec(query, sql.Named("newBlockNumber", newBlockNumber), sql.Named("previousBlockNumber", newBlockNumber-1))
	if results.Error != nil {
		return results.Error
	}
	return nil
}

func (p *PostgresBlockStore) CloneOperatorSharesForNewBlock(newBlockNumber uint64) error {
	query := `
		insert into operator_shares (operator, strategy, shares, block_number)
			select
				operator,
				strategy,
				shares,
				@newBlockNumber as block_number
			from operator_shares where block_number = @previousBlockNumber
	`
	results := p.Db.Exec(query, sql.Named("newBlockNumber", newBlockNumber), sql.Named("previousBlockNumber", newBlockNumber-1))
	if results.Error != nil {
		return results.Error
	}
	return nil
}

func (p *PostgresBlockStore) CloneStakerSharesForNewBlock(newBlockNumber uint64) error {
	query := `
		insert into staker_shares (staker, strategy, shares, block_number)
			select
				staker,
				strategy,
				shares,
				@newBlockNumber as block_number
			from staker_shares where block_number = @previousBlockNumber
	`
	results := p.Db.Exec(query, sql.Named("newBlockNumber", newBlockNumber), sql.Named("previousBlockNumber", newBlockNumber-1))
	if results.Error != nil {
		return results.Error
	}
	return nil
}

func (p *PostgresBlockStore) CloneDelegatedStakersForNewBlock(newBlockNumber uint64) error {
	query := `
		insert into delegated_stakers (staker, operator, block_number)
			select
				staker,
				operator,
				@blockNumber as block_number
			from delegated_stakers where block_number = @previousBlockNumber
	`
	results := p.Db.Exec(query, sql.Named("newBlockNumber", newBlockNumber), sql.Named("previousBlockNumber", newBlockNumber-1))
	if results.Error != nil {
		return results.Error
	}
	return nil
}

func (p *PostgresBlockStore) SetActiveRewardsForNewBlock(newBlockNumber uint64) error {
	// At the given block, we want to store all rewards that have not yet met their end_timestamp.
	//
	// Once end_timestamp < current_block.timestamp, the reward is no longer active and should not be included in the active_rewards table.
	query := `
		with current_block as (
			select * from blocks where number = @newBlockNumber limit 1
		),
		insert into active_rewards (avs, reward_hash, token, amount, strategy, multiplier, strategy_index, block_number, start_timestamp, end_timestamp, duration)
			select
				avs,
				reward_hash,
				token,
				amount,
				strategy,
				multiplier,
				strategy_index,
				@newBlockNumber as block_number,
				start_timestamp,
				end_timestamp,
				duration
			from active_reward_submissions
			where end_timestamp > current_block.timestamp
	`
	results := p.Db.Exec(query, sql.Named("newBlockNumber", newBlockNumber))
	if results.Error != nil {
		return results.Error
	}
	return nil
}

func (p *PostgresBlockStore) SetActiveRewardForAllForNewBlock(newBlockNumber uint64) error {
	// At the given block, we want to store all rewards that have not yet met their end_timestamp.
	//
	// Once end_timestamp < current_block.timestamp, the reward is no longer active and should not be included in the active_rewards table.
	query := `
		with current_block as (
			select * from blocks where number = @newBlockNumber limit 1
		),
		insert into active_reward_for_all (avs, reward_hash, token, amount, strategy, multiplier, strategy_index, block_number, start_timestamp, end_timestamp, duration)
			select
				avs,
				reward_hash,
				token,
				amount,
				strategy,
				multiplier,
				strategy_index,
				@newBlockNumber as block_number,
				start_timestamp,
				end_timestamp,
				duration
			from active_reward_for_all_submissions
			where end_timestamp > current_block.timestamp
	`
	results := p.Db.Exec(query, sql.Named("newBlockNumber", newBlockNumber))
	if results.Error != nil {
		return results.Error
	}
	return nil
}
