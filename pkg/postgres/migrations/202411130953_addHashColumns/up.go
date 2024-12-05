package _202411130953_addHashColumns

import (
	"database/sql"
	"github.com/Layr-Labs/sidecar/internal/config"
	"gorm.io/gorm"
)

type Migration struct {
}

func (m *Migration) Up(db *sql.DB, grm *gorm.DB, cfg *config.Config) error {
	queries := []string{
		`alter table reward_submissions add column transaction_hash varchar, add column log_index bigint`,
		`alter table blocks add column parent_hash varchar`,
		`alter table avs_operator_state_changes add column transaction_hash varchar`,
		`alter table staker_delegation_changes add column transaction_hash varchar`,
		`alter table submitted_distribution_roots add column transaction_hash varchar, add column log_index bigint`,
	}
	for _, query := range queries {
		if _, err := db.Exec(query); err != nil {
			return err
		}
	}
	return nil
}

func (m *Migration) GetName() string {
	return "202411130953_addHashColumns"
}
