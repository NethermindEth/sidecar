package _202409082234_stakerShare

import (
	"gorm.io/gorm"
)

type SqliteMigration struct {
}

func (m *SqliteMigration) Up(grm *gorm.DB) error {
	queries := []string{
		`create table if not exists staker_shares (
			staker TEXT NOT NULL,
			strategy TEXT NOT NULL,
			shares TEXT NOT NULL,
			block_number INTEGER NOT NULL,
			created_at DATETIME default current_timestamp,
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

func (m *SqliteMigration) GetName() string {
	return "202409082234_stakerShare"
}
