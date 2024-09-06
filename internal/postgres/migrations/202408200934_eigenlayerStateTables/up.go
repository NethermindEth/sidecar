package _202408200934_eigenlayerStateTables

import (
	"database/sql"
	"fmt"
	"github.com/Layr-Labs/sidecar/internal/postgres"
	"gorm.io/gorm"
)

type Migration struct {
}

func (m *Migration) Up(db *sql.DB, grm *gorm.DB) error {
	queries := []string{
		`create table if not exists avs_operator_changes (
			id serial primary key,
			operator varchar,
			avs varchar,
			registered boolean,
			transaction_hash varchar,
			transaction_index bigint,
			log_index bigint,
			block_number bigint,
			created_at timestamp with time zone default current_timestamp
		)
		`,
		`create index if not exists idx_avs_operator_changes_avs_operator on avs_operator_changes (avs, operator)`,
		`create index if not exists idx_avs_operator_changes_block on avs_operator_changes (block_number)`,
		`create table if not exists registered_avs_operators (
			operator varchar,
			avs varchar,
			block_number bigint,
			created_at timestamp with time zone default current_timestamp,
			unique(operator, avs, block_number)
		);
		`,
		`create index if not exists idx_registered_avs_operators_avs_operator on registered_avs_operators (avs, operator)`,
		`create index if not exists idx_registered_avs_operators_block on registered_avs_operators (block_number)`,
		/*
			`create table if not exists staker_share_changes (
				id serial primary key,
				staker varchar,
				strategy varchar,
				shares numeric,
				transaction_hash varchar,
				log_index bigint,
				block_number bigint,
				created_at timestamp with time zone default current_timestamp
			);
			`,
			`create index if not exists idx_staker_share_changes_staker_strat on staker_share_changes (staker, strategy)`,
			`create index if not exists idx_staker_share_changes_block on staker_share_changes (block_number)`,
			`create table if not exists staker_delegation_changes (
				id serial primary key,
				staker varchar,
				operator varchar,
				delegated boolean,
				transaction_hash varchar,
				log_index bigint,
				block_number bigint,
				created_at timestamp with time zone default current_timestamp
			);
			`,
			`create index if not exists idx_staker_delegation_changes_staker_operator on staker_delegation_changes (staker, operator)`,
			`create index if not exists idx_staker_delegation_changes_block on staker_delegation_changes (block_number)`,
			`create table if not exists active_reward_submissions (
				id serial primary key,
				avs varchar,
				reward_hash varchar,
				token varchar,
				amount numeric,
				strategy varchar,
				multiplier numeric,
				strategy_index bigint,
				transaction_hash varchar,
				log_index bigint,
				block_number bigint,
				start_timestamp timestamp,
				end_timestamp timestamp,
				duration bigint,
				created_at timestamp with time zone default current_timestamp
			);
			`,
			`create index if not exists idx_active_reward_submissions_avs on active_reward_submissions (avs)`,
			`create index if not exists idx_active_reward_submissions_block on active_reward_submissions (block_number)`,
			`create table if not exists active_reward_for_all_submissions (
				id serial primary key,
				avs varchar,
				reward_hash varchar,
				token varchar,
				amount numeric,
				strategy varchar,
				multiplier numeric,
				strategy_index bigint,
				transaction_hash varchar,
				log_index bigint,
				block_number bigint,
				start_timestamp timestamp,
				end_timestamp timestamp,
				duration bigint,
				created_at timestamp with time zone default current_timestamp
			);
			`,
			`create index if not exists idx_active_reward_for_all_submissions_avs on active_reward_for_all_submissions (avs)`,
			`create index if not exists idx_active_reward_for_all_submissions_block on active_reward_for_all_submissions (block_number)`,
			`create table if not exists staker_shares (
				staker varchar,
				strategy varchar,
				shares numeric,
				block_number bigint,
				created_at timestamp with time zone default current_timestamp,
				unique(staker, strategy, block_number)
			)
			`,
			`create index if not exists idx_staker_shares_staker_strategy on staker_shares (staker, strategy)`,
			`create index if not exists idx_staker_shares_block on staker_shares (block_number)`,
			`create table if not exists delegated_stakers (
				staker varchar,
				operator varchar,
				block_number bigint,
				created_at timestamp with time zone default current_timestamp,
				unique(staker, operator, block_number)
			)`,
			`create index if not exists idx_delegated_stakers_staker_operator on delegated_stakers (staker, operator)`,
			`create index if not exists idx_delegated_stakers_block on delegated_stakers (block_number)`,
			`create table if not exists active_rewards (
				avs varchar,
				reward_hash varchar,
				token varchar,
				amount numeric,
				strategy varchar,
				multiplier numeric,
				strategy_index bigint,
				block_number bigint,
				start_timestamp timestamp,
				end_timestamp timestamp,
				duration bigint,
				created_at timestamp with time zone default current_timestamp
			)`,
			`create index if not exists idx_active_rewards_avs on active_rewards (avs)`,
			`create index if not exists idx_active_rewards_block on active_rewards (block_number)`,
			`create table if not exists active_reward_for_all (
				avs varchar,
				reward_hash varchar,
				token varchar,
				amount numeric,
				strategy varchar,
				multiplier numeric,
				strategy_index bigint,
				block_number bigint,
				start_timestamp timestamp,
				end_timestamp timestamp,
				duration bigint,
				created_at timestamp with time zone default current_timestamp
			)`,
			`create index if not exists idx_active_reward_for_all_avs on active_reward_for_all (avs)`,
			`create index if not exists idx_active_reward_for_all_block on active_reward_for_all (block_number)`,
		*/
	}

	// Wrap the queries in a transaction so they all create or fail atomically
	_, err := postgres.WrapTxAndCommit[interface{}](func(tx *gorm.DB) (interface{}, error) {
		p, err := tx.DB()
		if err != nil {
			return nil, err
		}
		for _, query := range queries {
			_, err := p.Exec(query)
			if err != nil {
				fmt.Printf("Failed to execute query: %s\n", query)
				return nil, err
			}
		}
		return nil, nil
	}, grm, nil)
	return err
}

func (m *Migration) GetName() string {
	return "202408200934_eigenlayerStateTables"
}
