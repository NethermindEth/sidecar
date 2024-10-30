package _202409080918_staterootTable

import (
	"database/sql"
	"gorm.io/gorm"
)

type Migration struct {
}

func (m *Migration) Up(db *sql.DB, grm *gorm.DB) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS state_roots (
    		eth_block_number bigint not null,
    		eth_block_hash varchar not null,
    		state_root varchar not null,
    		created_at timestamp with time zone DEFAULT current_timestamp,
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

func (m *Migration) GetName() string {
	return "202409080918_staterootTable"
}
