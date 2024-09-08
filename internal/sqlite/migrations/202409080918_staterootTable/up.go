package _202409080918_staterootTable

import (
	"gorm.io/gorm"
)

type SqliteMigration struct {
}

func (m *SqliteMigration) Up(grm *gorm.DB) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS state_roots (
    		eth_block_number INTEGER,
    		eth_block_hash TEXT,
    		state_root TEXT,
    		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    		unique(eth_block_hash)
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
	return "202409080918_staterootTable"
}
