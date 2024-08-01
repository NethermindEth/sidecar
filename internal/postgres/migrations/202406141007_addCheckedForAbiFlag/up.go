package _202406141007_addCheckedForAbiFlag

import (
	"database/sql"
	"gorm.io/gorm"
)

type Migration struct {
}

func (m *Migration) Up(db *sql.DB, grm *gorm.DB) error {
	queries := []string{
		`alter table contracts add column checked_for_abi boolean`,
		`update contracts set checked_for_abi = true where length(contract_abi) > 0 or matching_contract_address != ''`,
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
	return "202406141007_addCheckedForAbiFlag"
}
