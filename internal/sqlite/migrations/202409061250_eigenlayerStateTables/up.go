package _202409061250_eigenlayerStateTables

import (
	"fmt"
	"gorm.io/gorm"
)

type SqliteMigration struct {
}

func (m *SqliteMigration) Up(grm *gorm.DB) error {
	queries := []string{
		`create table if not exists registered_avs_operators (
			operator TEXT NOT NULL,
			avs TEXT NOT NULL,
			block_number INTEGER NOT NULL,
			created_at DATETIME default current_timestamp,
			unique(operator, avs, block_number)
		);
		`,
	}

	for _, query := range queries {
		if res := grm.Exec(query); res.Error != nil {
			fmt.Printf("Failed to execute query: %s\n", query)
			return res.Error
		}
	}
	return nil
}

func (m *SqliteMigration) GetName() string {
	return "202409061250_eigenlayerStateTables"
}
