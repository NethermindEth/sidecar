package _202405201503_fixTransactionHashConstraint

import (
	"database/sql"
	"gorm.io/gorm"
)

type Migration struct {
}

func (m *Migration) Up(db *sql.DB, grm *gorm.DB) error {

	queries := []string{
		"ALTER TABLE transaction_logs DROP CONSTRAINT transaction_logs_transaction_hash_fkey",
		"alter table transactions drop constraint transactions_transaction_hash_key",
		"alter table transactions add constraint transactions_transaction_hash_sequence_id_key unique (transaction_hash, block_sequence_id)",
		"alter table transaction_logs add constraint fk_transaction_hash_sequence_id_key foreign key (transaction_hash, block_sequence_id) references transactions(transaction_hash, block_sequence_id) on delete cascade",
	}

	for _, query := range queries {
		if err := grm.Exec(query).Error; err != nil {
			return err
		}
	}
	return nil
}

func (m *Migration) GetName() string {
	return "202405201503_fixTransactionHashConstraint"
}
