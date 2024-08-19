package _202406031946_addSerialIdToContracts

import (
	"database/sql"
	"gorm.io/gorm"
)

type Migration struct {
}

func (m *Migration) Up(db *sql.DB, grm *gorm.DB) error {
	query := `alter table contracts add column id serial`

	_, err := db.Exec(query)
	if err != nil {
		return err
	}
	return nil
}

func (m *Migration) GetName() string {
	return "202406031946_addSerialIdToContracts"
}
