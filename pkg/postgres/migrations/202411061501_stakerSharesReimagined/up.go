package _202411061501_stakerSharesReimagined

import (
	"database/sql"
	"github.com/Layr-Labs/sidecar/internal/config"
	"gorm.io/gorm"
)

type Migration struct {
}

func (m *Migration) Up(db *sql.DB, grm *gorm.DB, cfg *config.Config) error {
	queries := []string{
		`drop table if exists staker_shares`,
		`create table if not exists staker_shares (
			staker varchar not null,
			strategy varchar not null,
			shares numeric not null,
			strategy_index bigint not null,
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
	return "202411061501_stakerSharesReimagined"
}
