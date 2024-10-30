package _202410301449_generatedRewardsSnapshots

import (
	"database/sql"
	"gorm.io/gorm"
)

type Migration struct {
}

// processing, complete, failed

func (m *Migration) Up(db *sql.DB, grm *gorm.DB) error {
	queries := []string{
		`CREATE TYPE reward_snapshot_status AS ENUM ('processing', 'complete', 'failed');`,
		`create table if not exists generated_rewards_snapshots (
			id serial primary key,
			snapshot_date varchar not null,
			status reward_snapshot_status not null,
			created_at timestamp with time zone default now(),
    		updated_at timestamp with time zone,
    		unique(snapshot_date)
		)
		`,
	}
	for _, query := range queries {
		if _, err := db.Exec(query); err != nil {
			return err
		}
	}
	return nil
}

func (m *Migration) GetName() string {
	return "202410301449_generatedRewardsSnapshots"
}
