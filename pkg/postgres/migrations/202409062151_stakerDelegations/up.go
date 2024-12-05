package _202409062151_stakerDelegations

import (
	"database/sql"
	"github.com/Layr-Labs/sidecar/internal/config"
	"gorm.io/gorm"
)

type Migration struct {
}

func (m *Migration) Up(db *sql.DB, grm *gorm.DB, cfg *config.Config) error {
	queries := []string{
		`create table if not exists delegated_stakers (
			staker varchar not null,
			operator varchar not null,
			block_number bigint not null,
			created_at timestamp with time zone DEFAULT current_timestamp,
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

func (m *Migration) GetName() string {
	return "202409062151_stakerDelegations"
}
