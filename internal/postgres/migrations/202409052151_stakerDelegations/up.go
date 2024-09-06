package _202409052151_stakerDelegations

import (
	"database/sql"
	"gorm.io/gorm"
)

type Migration struct {
}

func (m *Migration) Up(db *sql.DB, grm *gorm.DB) error {
	queries := []string{
		`create table if not exists staker_delegation_changes (
				staker varchar,
				operator varchar,
				delegated boolean,
				transaction_hash varchar,
				log_index bigint,
				transaction_index bigint,
				block_number bigint,
				created_at timestamp with time zone default current_timestamp,
				unique(staker, operator, log_index, block_number)
			);`,
		`create index if not exists idx_staker_delegation_changes_staker_operator on staker_delegation_changes (staker, operator)`,
		`create index if not exists idx_staker_delegation_changes_block on staker_delegation_changes (block_number)`,
		`create table if not exists delegated_stakers (
				staker varchar,
				operator varchar,
				block_number bigint,
				created_at timestamp with time zone default current_timestamp,
				unique(staker, operator, block_number)
			)`,
		`create index if not exists idx_delegated_stakers_staker_operator on delegated_stakers (staker, operator)`,
		`create index if not exists idx_delegated_stakers_block on delegated_stakers (block_number)`,
	}
	for _, query := range queries {
		if _, err := db.Exec(query); err != nil {
			return err
		}
	}
	return nil
}

func (m *Migration) GetName() string {
	return "202409052151_stakerDelegations"
}
