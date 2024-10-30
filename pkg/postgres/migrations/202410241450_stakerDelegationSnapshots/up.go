package _202410241450_stakerDelegationSnapshots

import (
	"database/sql"
	"gorm.io/gorm"
)

type Migration struct {
}

func (m *Migration) Up(db *sql.DB, grm *gorm.DB) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS staker_delegation_snapshots (
				staker varchar not null,
				operator varchar not null,
				snapshot date not null
			)
		`,
		`create index idx_staker_delegation_snapshots_operator_snapshot on staker_delegation_snapshots (operator, snapshot)`,
	}
	for _, query := range queries {
		if _, err := db.Exec(query); err != nil {
			return err
		}
	}
	return nil
}

func (m *Migration) GetName() string {
	return "202410241450_stakerDelegationSnapshots"
}
