package _202409062151_stakerDelegations

import (
	"gorm.io/gorm"
)

type SqliteMigration struct {
}

func (m *SqliteMigration) Up(grm *gorm.DB) error {
	queries := []string{
		`create table if not exists delegated_stakers (
			staker TEXT NOT NULL,
			operator TEXT NOT NULL,
			block_number INTEGER NOT NULL,
			created_at DATETIME default current_timestamp,
			unique(staker, operator, block_number)
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
	return "202409062151_stakerDelegations"
}
