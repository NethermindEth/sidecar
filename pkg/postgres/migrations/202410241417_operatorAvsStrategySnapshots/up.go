package _202410241417_operatorAvsStrategySnapshots

import (
	"database/sql"
	"gorm.io/gorm"
)

type Migration struct {
}

func (m *Migration) Up(db *sql.DB, grm *gorm.DB) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS operator_avs_strategy_snapshots (
				operator varchar not null,
				avs varchar not null,
				strategy varchar not null,
				snapshot DATE NOT NULL
			);
		`,
		`CREATE INDEX IF NOT EXISTS idx_operator_avs_strategy_snapshots_avs_strat_snap ON operator_avs_strategy_snapshots (avs, strategy, snapshot);`,
	}

	for _, query := range queries {
		if err := grm.Exec(query).Error; err != nil {
			return err
		}
	}
	return nil
}

func (m *Migration) GetName() string {
	return "202410241417_operatorAvsStrategySnapshots"
}
