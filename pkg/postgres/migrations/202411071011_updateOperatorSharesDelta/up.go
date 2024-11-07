package _202411071011_updateOperatorSharesDelta

import (
	"database/sql"
	"gorm.io/gorm"
)

type Migration struct {
}

func (m *Migration) Up(db *sql.DB, grm *gorm.DB) error {
	queries := []string{
		`alter table operator_share_deltas add column staker varchar`,
		`drop table operator_shares`,
		`create table if not exists operator_shares (
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
		_, err := db.Exec(query)
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *Migration) GetName() string {
	return "202411071011_updateOperatorSharesDelta"
}
