package _202501241111_addIndexesForRpcFunctions

import (
	"database/sql"
	"github.com/Layr-Labs/sidecar/internal/config"
	"gorm.io/gorm"
)

type Migration struct {
}

func (m *Migration) Up(db *sql.DB, grm *gorm.DB, cfg *config.Config) error {
	queries := []string{
		`create index concurrently if not exists idx_gold_table_snapshot on gold_table(snapshot)`,
		`create index concurrently if not exists idx_gold_table_earner on gold_table(earner)`,
		`create index concurrently if not exists idx_gold_table_earner_tokens on gold_table(earner, token)`,
		`create index concurrently if not exists idx_staker_delegation_changes_operator on staker_delegation_changes(operator)`,
		`create index concurrently if not exists idx_staker_share_deltas_staker_strategy on staker_share_deltas(staker, strategy)`,
		`create index concurrently if not exists idx_staker_delegation_changes_staker on staker_delegation_changes(staker)`,
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
	return "202501241111_addIndexesForRpcFunctions"
}
