package _202411120947_disabledDistributionRoots

import (
	"database/sql"
	"gorm.io/gorm"
)

type Migration struct {
}

func (m *Migration) Up(db *sql.DB, grm *gorm.DB) error {
	query := `
		create table if not exists disabled_distribution_roots (
			root_index bigint not null,
			log_index bigint not null,
			transaction_hash varchar not null,
			block_number bigint not null
		);	
	`
	_, err := db.Exec(query)
	return err
}

func (m *Migration) GetName() string {
	return "202411120947_disabledDistributionRoots"
}
