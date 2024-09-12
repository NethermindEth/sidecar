package _202409101540_rewardSubmissions

import (
	"gorm.io/gorm"
)

type SqliteMigration struct {
}

func (m *SqliteMigration) Up(grm *gorm.DB) error {
	query := `
		create table if not exists reward_submissions (
			avs TEXT NOT NULL,
			reward_hash TEST NOT NULL,
			token TEXT NOT NULL,
			amount TEXT NOT NULL,
			strategy TEXT NOT NULL,
			strategy_index INTEGER NOT NULL,
			multiplier TEXT NOT NULL,
			start_timestamp DATETIME NOT NULL,
			end_timestamp DATETIME NOT NULL,
			duration INTEGER NOT NULL,
			is_for_all INTEGER DEFAULT 0,
			block_number INTEGER NOT NULL,
			unique(reward_hash, strategy, block_number)
		);
	`
	if err := grm.Exec(query).Error; err != nil {
		return err
	}
	return nil
}

func (m *SqliteMigration) GetName() string {
	return "202409101540_rewardSubmissions"
}
