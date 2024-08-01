package _202406251426_addTransactionIndexes

import (
	"database/sql"
	"gorm.io/gorm"
)

type Migration struct {
}

func (m *Migration) Up(db *sql.DB, grm *gorm.DB) error {
	queries := []string{
		`create index idx_transactions_to_address on transactions(to_address);`,
		`create index idx_transactions_from_address on transactions(from_address);`,
		`create index idx_transactions_bytecode_hash on transactions(bytecode_hash);`,
		`create index idx_transactions_block_number on transactions(block_number);`,
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
	return "202406251426_addTransactionIndexes"
}
