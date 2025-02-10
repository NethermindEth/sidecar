package _202502100846_goldTableRewardHashIndex

import (
	"database/sql"
	"github.com/Layr-Labs/sidecar/internal/config"
	"gorm.io/gorm"
)

type Migration struct {
}

func (m *Migration) Up(db *sql.DB, grm *gorm.DB, cfg *config.Config) error {
	queries := []string{
		`create index if not exists idx_gold_table_reward_hash on gold_table (reward_hash)`,
		`create index if not exists idx_combined_rewards_avs_reward_hash on combined_rewards (avs, reward_hash, block_number)`,
		`create index if not exists idx_combined_rewards_block_number on combined_rewards (block_number)`,
	}
	for _, query := range queries {
		res := grm.Exec(query)
		if res.Error != nil {
			return res.Error
		}
	}
	return nil
}

func (m *Migration) GetName() string {
	return "202502100846_goldTableRewardHashIndex"
}
