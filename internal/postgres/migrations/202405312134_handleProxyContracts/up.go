package _202405312134_handleProxyContracts

import (
	"database/sql"
	"gorm.io/gorm"
)

type Migration struct {
}

func (m *Migration) Up(db *sql.DB, grm *gorm.DB) error {
	queries := []string{
		`create table if not exists proxy_contracts (
			block_number bigint not null,
			contract_address varchar(255) not null,
			proxy_contract_address varchar(255) not null,
			created_at timestamp not null default current_timestamp,
    		updated_at timestamp with time zone DEFAULT NULL,
			deleted_at timestamp with time zone DEFAULT NULL
		)`,
		`create unique index idx_unique_proxy_contract on proxy_contracts(block_number, contract_address, proxy_contract_address)`,
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
	return "202405312134_handleProxyContracts"
}
