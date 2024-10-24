package _202409082234_stakerShare

import (
	"database/sql"
	"gorm.io/gorm"
)

type Migration struct {
}

func (m *Migration) Up(db *sql.DB, grm *gorm.DB) error {
	queries := []string{
		`create table if not exists staker_shares (
			staker varchar not null,
			strategy varchar not null,
			shares numeric not null,
			block_number bigint not null,
			created_at timestamp with time zone DEFAULT current_timestamp,
			unique (staker, strategy, block_number)
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
	return "202409082234_stakerShare"
}
