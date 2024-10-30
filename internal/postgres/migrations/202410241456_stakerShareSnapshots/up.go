package _202410241456_stakerShareSnapshots

import (
	"database/sql"
	"gorm.io/gorm"
)

type Migration struct {
}

func (m *Migration) Up(db *sql.DB, grm *gorm.DB) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS staker_share_snapshots (
			staker varchar not null,
			strategy varchar not null,
			shares numeric not null,
			snapshot date not null
		)
		`,
		`create index idx_staker_share_snapshots_staker_strategy_snapshot on staker_share_snapshots (staker, strategy, snapshot)`,
		`create index idx_staker_share_snapshots_strategy_snapshot on staker_share_snapshots (strategy, snapshot)`,
	}
	for _, query := range queries {
		if _, err := db.Exec(query); err != nil {
			return err
		}
	}
	return nil
}

func (m *Migration) GetName() string {
	return "202410241456_stakerShareSnapshots"
}
