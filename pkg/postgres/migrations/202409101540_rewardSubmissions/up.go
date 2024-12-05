package _202409101540_rewardSubmissions

import (
	"database/sql"
	"github.com/Layr-Labs/sidecar/internal/config"
	"gorm.io/gorm"
)

type Migration struct {
}

func (m *Migration) Up(db *sql.DB, grm *gorm.DB, cfg *config.Config) error {
	query := `
		create table if not exists reward_submissions (
			avs varchar not null,
			reward_hash varchar not null,
			token varchar not null,
			amount numeric not null,
			strategy varchar not null,
			strategy_index integer not null,
			multiplier numeric(78) not null,
			start_timestamp timestamp(6) not null,
			end_timestamp timestamp(6) not null,
			duration bigint not null,
			is_for_all boolean not null default false,
			block_number bigint not null,
			reward_type varchar not null,
			unique(reward_hash, strategy, block_number)
		);
	`
	if err := grm.Exec(query).Error; err != nil {
		return err
	}
	return nil
}

func (m *Migration) GetName() string {
	return "202409101540_rewardSubmissions"
}
