package _202409051720_operatorShareChanges

import (
	"database/sql"
	"gorm.io/gorm"
)

type Migration struct {
}

func (m *Migration) Up(db *sql.DB, grm *gorm.DB) error {
	queries := []string{
		`create table if not exists operator_share_changes (
				id serial primary key,
				operator varchar,
				strategy varchar,
				shares numeric,
				transaction_hash varchar,
				transaction_index bigint,
				log_index bigint,
				block_number bigint,
				created_at timestamp with time zone default current_timestamp,
				unique(operator, strategy, transaction_hash, log_index)
			)
			`,
		`create index if not exists idx_operator_share_changes_operator_strat on operator_share_changes (operator, strategy)`,
		`create index if not exists idx_operator_share_changes_block on operator_share_changes (block_number)`,
		`create table if not exists operator_shares (
				operator varchar,
				strategy varchar,
				shares numeric,
				block_number bigint,
				created_at timestamp with time zone default current_timestamp,
				unique (operator, strategy, block_number)
			)`,
		`create index if not exists idx_operator_shares_operator_strategy on operator_shares (operator, strategy)`,
		`create index if not exists idx_operator_shares_block on operator_shares (block_number)`,
	}
	for _, query := range queries {
		if _, err := db.Exec(query); err != nil {
			return err
		}
	}
	return nil
}

func (m *Migration) GetName() string {
	return "202409051720_operatorShareChanges"
}
