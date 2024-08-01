package _202405151523_addTransactionToFrom

import (
	"database/sql"
	"gorm.io/gorm"
)

type Migration struct {
}

func (m *Migration) Up(db *sql.DB, grm *gorm.DB) error {
	query := `ALTER TABLE transactions ADD COLUMN from_address varchar(255) NOT NULL, ADD COLUMN to_address varchar(255) DEFAULT NULL`

	result := grm.Exec(query)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (m *Migration) GetName() string {
	return "202405151523_addTransactionToFrom"
}
