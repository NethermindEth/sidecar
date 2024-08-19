package _202405171056_unverifiedContracts

import (
	"database/sql"
	"gorm.io/gorm"
)

type Migration struct {
}

func (m *Migration) Up(db *sql.DB, grm *gorm.DB) error {
	query := `CREATE TABLE IF NOT EXISTS unverified_contracts (
		contract_address varchar(255) PRIMARY KEY,
		created_at timestamp with time zone DEFAULT current_timestamp,
		updated_at timestamp with time zone DEFAULT NULL,
		deleted_at timestamp with time zone DEFAULT NULL,
		UNIQUE(contract_address)
	)`
	res := grm.Exec(query)
	if res.Error != nil {
		return res.Error
	}
	return nil
}

func (m *Migration) GetName() string {
	return "202405171056_unverifiedContracts"
}
