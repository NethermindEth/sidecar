package _202407121407_updateProxyContractIndex

import (
	"database/sql"
	"gorm.io/gorm"
)

type Migration struct {
}

func (m *Migration) Up(db *sql.DB, grm *gorm.DB) error {
	queries := []string{
		`create unique index idx_uniq_proxy_contract on proxy_contracts(block_number, contract_address)`,
		`drop index idx_unique_proxy_contract`,
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
	return "202407121407_updateProxyContractIndex"
}
