package _202411131200_eigenStateModelConstraints

import (
	"database/sql"
	"gorm.io/gorm"
)

type Migration struct {
}

func (m *Migration) Up(db *sql.DB, grm *gorm.DB) error {
	queries := []string{
		`alter table disabled_distribution_roots add constraint uniq_disabled_distribution_root unique (transaction_hash, log_index, block_number)`,
		`alter table operator_share_deltas add constraint uniq_operator_share unique (transaction_hash, log_index, block_number, operator, staker, strategy)`,
		`alter table reward_submissions add constraint uniq_reward_submission unique (transaction_hash, log_index, block_number, reward_hash, strategy_index)`,
		`alter table reward_submissions drop constraint reward_submissions_reward_hash_strategy_block_number_key`,
		`alter table staker_delegation_changes add constraint uniq_staker_delegation_change unique (transaction_hash, log_index, block_number)`,
		`alter table staker_share_deltas add constraint uniq_staker_share_delta unique (transaction_hash, log_index, block_number, staker, strategy, strategy_index)`,
		`alter table submitted_distribution_roots add constraint uniq_submitted_distribution_root unique (transaction_hash, log_index, block_number)`,
	}

	for _, query := range queries {
		if _, err := db.Exec(query); err != nil {
			return err
		}
	}
	return nil
}

func (m *Migration) GetName() string {
	return "202411131200_eigenStateModelConstraints"
}
