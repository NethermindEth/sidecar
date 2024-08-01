package _202407101440_addOperatorRestakedStrategiesTable

import (
	"database/sql"
	"gorm.io/gorm"
)

type Migration struct {
}

func (m *Migration) Up(db *sql.DB, grm *gorm.DB) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS operator_restaked_strategies (
			id serial PRIMARY KEY,
			block_number bigint NOT NULL,
			operator varchar NOT NULL,
			avs varchar NOT NULL,
			strategy varchar NOT NULL,
			created_at timestamp with time zone DEFAULT current_timestamp,
			updated_at timestamp with time zone DEFAULT NULL,
			deleted_at timestamp with time zone DEFAULT NULL
		)`,
		`create unique index idx_unique_operator_restaked_strategies on operator_restaked_strategies(block_number, operator, avs, strategy)`,
	}

	for _, query := range queries {
		_, err := db.Exec(query)
		if err != nil {
			return err
		}
	}
	return nil
}

func (m *Migration) GetName() string {
	return "202407101440_addOperatorRestakedStrategiesTable"
}
