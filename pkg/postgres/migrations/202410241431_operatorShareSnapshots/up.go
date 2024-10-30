package _202410241431_operatorShareSnapshots

import (
	"database/sql"
	"gorm.io/gorm"
)

type Migration struct {
}

func (m *Migration) Up(db *sql.DB, grm *gorm.DB) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS operator_share_snapshots (
			operator varchar not null,
			strategy varchar not null,
			shares numeric not null,
			snapshot date not null
		)`,
	}
	for _, query := range queries {
		if _, err := db.Exec(query); err != nil {
			return err
		}
	}
	return nil
}

func (m *Migration) GetName() string {
	return "202410241431_operatorShareSnapshots"
}
