package _202406030920_addCheckedForProxyFlag

import (
	"database/sql"
	"gorm.io/gorm"
)

type Migration struct {
}

func (m *Migration) Up(db *sql.DB, grm *gorm.DB) error {
	query := `alter table contracts add column checked_for_proxy boolean not null default false;`

	_, err := db.Exec(query)
	if err != nil {
		return err
	}
	return nil
}

func (m *Migration) GetName() string {
	return "202406030920_addCheckedForProxyFlag"
}
