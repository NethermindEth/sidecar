package _202409101144_submittedDistributionRoot

import (
	"gorm.io/gorm"
)

type SqliteMigration struct {
}

func (m *SqliteMigration) Up(grm *gorm.DB) error {
	queries := []string{
		`create table if not exists submitted_distribution_roots (
			root text not null,
			root_index integer not null,
			rewards_calculation_end text not null,
			rewards_calculation_end_unit text not null,
			activated_at text not null,
			activated_at_unit text not null,
			created_at_block_number integer not null,
			block_number integer not null
		)`,
	}
	for _, query := range queries {
		if res := grm.Exec(query); res.Error != nil {
			return res.Error
		}
	}
	return nil
}

func (m *SqliteMigration) GetName() string {
	return "202409101144_submittedDistributionRoot"
}
