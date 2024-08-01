package _202405300925_addUniqueBlockConstraint

import (
	"database/sql"
	"gorm.io/gorm"
)

type Migration struct {
}

func (m *Migration) Up(db *sql.DB, grm *gorm.DB) error {
	query := `alter table blocks add constraint blocks_unique_block_number_hash unique (number, hash)`

	res := grm.Exec(query)
	if res.Error != nil {
		return res.Error
	}
	return nil
}

func (m *Migration) GetName() string {
	return "202405300925_addUniqueBlockConstraint"
}
