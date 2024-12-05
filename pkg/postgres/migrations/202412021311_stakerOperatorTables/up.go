package _202412021311_stakerOperatorTables

import (
	"database/sql"
	"github.com/Layr-Labs/sidecar/internal/config"
	"gorm.io/gorm"
)

type Migration struct {
}

func (m *Migration) Up(db *sql.DB, grm *gorm.DB, cfg *config.Config) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS staker_operator (
			earner text,
			operator text,
			reward_type text,
			avs text,
			token text,
			strategy text,
			multiplier numeric(78),
			shares numeric,
			amount numeric,
			reward_hash text,
			snapshot date
		);`,
		`alter table staker_operator add constraint uniq_staker_operator unique (earner, operator, snapshot, reward_hash, strategy, reward_type);`,
	}

	for _, query := range queries {
		if err := grm.Exec(query).Error; err != nil {
			return err
		}
	}
	return nil
}

func (m *Migration) GetName() string {
	return "202412021311_stakerOperatorTables"
}
