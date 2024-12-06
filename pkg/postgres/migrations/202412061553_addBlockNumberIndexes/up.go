package _202412061553_addBlockNumberIndexes

import (
	"database/sql"
	"fmt"
	"github.com/Layr-Labs/sidecar/internal/config"
	"gorm.io/gorm"
)

type Migration struct {
}

func (m *Migration) Up(db *sql.DB, grm *gorm.DB, cfg *config.Config) error {
	tables := []string{
		"avs_operator_state_changes",
		"operator_restaked_strategies",
		"operator_share_deltas",
		"reward_submissions",
		"staker_delegation_changes",
		"staker_share_deltas",
		"state_roots",
		"submitted_distribution_roots",
		"transaction_logs",
		"transactions",
	}

	for _, table := range tables {
		var query string
		if table != "state_roots" {
			query = fmt.Sprintf("create index if not exists %s_block_number_idx on %s (block_number)", table, table)
		} else {
			query = fmt.Sprintf("create index if not exists %s_eth_block_number_idx on %s (eth_block_number)", table, table)
		}
		result := grm.Exec(query)
		if result.Error != nil {
			return result.Error
		}
	}
	return nil
}

func (m *Migration) GetName() string {
	return "202412061553_addBlockNumberIndexes"
}
