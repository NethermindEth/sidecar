package _202412021311_stakerOperatorTables

import (
	"database/sql"
	"gorm.io/gorm"
)

type Migration struct {
}

func (m *Migration) Up(db *sql.DB, grm *gorm.DB) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS sot_staker_strategy_payouts (
			reward_hash varchar NOT NULL,
			snapshot TIMESTAMP NOT NULL,
			token varchar NOT NULL,
			tokens_per_day double precision NOT NULL,
			avs varchar NOT NULL,
			strategy varchar NOT NULL,
			multiplier numeric NOT NULL,
			reward_type varchar NOT NULL,
			operator varchar NOT NULL,
			staker varchar NOT NULL,
			shares numeric NOT NULL,
			staker_tokens numeric NOT NULL,
			staker_strategy_weight numeric NOT NULL,
			staker_total_strategy_weight numeric NOT NULL,
			staker_strategy_proportion numeric NOT NULL,
			staker_strategy_tokens numeric NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS sot_operator_strategy_rewards (
			reward_hash varchar NOT NULL,
			snapshot TIMESTAMP NOT NULL,
			token varchar NOT NULL,
			tokens_per_day double precision NOT NULL,
			avs varchar NOT NULL,
			strategy varchar NOT NULL,
			multiplier numeric NOT NULL,
			reward_type varchar NOT NULL,
			operator varchar NOT NULL,
			shares numeric NOT NULL,
			operator_tokens numeric NOT NULL,
			operator_strategy_weight numeric NOT NULL,
			operator_total_strategy_weight numeric NOT NULL,
			operator_strategy_proportion numeric NOT NULL,
			operator_strategy_tokens numeric NOT NULL
		)`,
	}

	for _, query := range queries {
		if err := grm.Exec(query).Error; err != nil {
			return err
		}
	}
	return nil
}

func (m *Migration) GetName() string {
	return "202412021311_stakerOperatorTables"
}
