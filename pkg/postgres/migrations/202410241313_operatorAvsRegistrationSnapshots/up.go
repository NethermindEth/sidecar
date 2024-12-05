package _202410241313_operatorAvsRegistrationSnapshots

import (
	"database/sql"
	"github.com/Layr-Labs/sidecar/internal/config"
	"gorm.io/gorm"
)

type Migration struct {
}

func (m *Migration) Up(db *sql.DB, grm *gorm.DB, cfg *config.Config) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS operator_avs_registration_snapshots (
			avs varchar not null,
			operator varchar not null,
			snapshot date not null
		)`,
		`CREATE INDEX IF NOT EXISTS idx_operator_avs_registration_snapshots_avs_snapshot ON operator_avs_registration_snapshots (avs, snapshot)`,
	}

	for _, query := range queries {
		if err := grm.Exec(query).Error; err != nil {
			return err
		}
	}
	return nil
}

func (m *Migration) GetName() string {
	return "202410241313_operatorAvsRegistrationSnapshots"
}
