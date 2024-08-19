package _202405312008_indexTransactionContractAddress

import (
	"database/sql"
	"gorm.io/gorm"
)

type Migration struct {
}

func (m *Migration) Up(db *sql.DB, grm *gorm.DB) error {
	queries := []string{
		`alter table transactions add column contract_address varchar(255) default null`,
		`create index transactions_contract_address on transactions(contract_address)`,
		`alter table transactions add column bytecode_hash varchar(64) default null`,
		`alter table contracts add column bytecode_hash varchar(64) default null`,
		`alter table contracts add column verified boolean default false`,
		`update contracts set verified = true`,
		`alter table contracts add column matching_contract_address varchar(255) default null`,
		`alter table contracts alter column contract_abi drop not null`,
		`insert into contracts(contract_address, verified) select distinct(contract_address), false from unverified_contracts where contract_address not in (select contract_address from contracts)`,
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
	return "202405312008_indexTransactionContractAddress"
}
