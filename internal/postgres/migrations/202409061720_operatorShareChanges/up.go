package _202409061720_operatorShareChanges

import (
	"database/sql"
	"gorm.io/gorm"
)

type Migration struct {
}

func (m *Migration) Up(db *sql.DB, grm *gorm.DB) error {
	queries := []string{
		`create table if not exists operator_shares (
			operator varchar not null,
			strategy varchar not null,
			shares NUMERIC not null,
			block_number bigint not null,
			created_at timestamp with time zone DEFAULT current_timestamp,
			unique (operator, strategy, block_number)
		)`,
		`create table if not exists operator_share_deltas (
    		operator varchar not null,
			strategy varchar not null,
			shares numeric not null,
			transaction_hash varchar not null,
			log_index bigint not null,
			block_time timestamp not null,
			block_date varchar not null,
			block_number bigint not null
		)`,
	}
	for _, query := range queries {
		if res := grm.Exec(query); res.Error != nil {
			return res.Error
		}
	}
	return nil
}

func (m *Migration) GetName() string {
	return "202409061720_operatorShareChanges"
}
