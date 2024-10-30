package _202409101144_submittedDistributionRoot

import (
	"database/sql"
	"gorm.io/gorm"
)

type Migration struct {
}

func (m *Migration) Up(db *sql.DB, grm *gorm.DB) error {
	queries := []string{
		`create table if not exists submitted_distribution_roots (
			root varchar not null,
			root_index bigint not null,
			rewards_calculation_end timestamp without time zone not null,
			rewards_calculation_end_unit varchar not null,
			activated_at timestamp without time zone not null,
			activated_at_unit varchar not null,
			created_at_block_number bigint not null,
			block_number bigint not null
		)`,
	}
	for _, query := range queries {
		if res := grm.Exec(query); res.Error != nil {
			return res.Error
		}
	}
	return nil
}

func (m *Migration) GetName() string {
	return "202409101144_submittedDistributionRoot"
}
