package _202406051937_addBytecodeIndex

import (
	"database/sql"
	"gorm.io/gorm"
)

type Migration struct {
}

func (m *Migration) Up(db *sql.DB, grm *gorm.DB) error {
	queries := []string{
		`create index idx_bytecode_hash on contracts (bytecode_hash)`,
		`create index idx_proxy_contract_proxy_contract_address on proxy_contracts (proxy_contract_address)`,
		`create index idx_proxy_contract_contract_address on proxy_contracts (contract_address)`,
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
	return "202406051937_addBytecodeIndex"
}
