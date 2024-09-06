package _202409061121_removeSequenceId

import (
	"database/sql"
	"gorm.io/gorm"
)

type Migration struct {
}

func (m *Migration) Up(db *sql.DB, grm *gorm.DB) error {
	queries := []string{
		// drop constraints
		`alter table transactions drop constraint transactions_transaction_hash_sequence_id_key`,
		`alter table transaction_logs drop constraint fk_transaction_hash_sequence_id_key`,
		`alter table transaction_logs drop constraint transaction_logs_block_sequence_id_fkey`,
		`alter table transactions drop constraint transactions_block_sequence_id_fkey`,
		// re-add constraints
		`alter table transactions add constraint transactions_transaction_hash_block_number_key unique (transaction_hash, block_number)`,
		`alter table transaction_logs ADD CONSTRAINT fk_transaction_hash_block_number_key FOREIGN KEY (transaction_hash, block_number) REFERENCES public.transactions(transaction_hash, block_number) ON DELETE CASCADE;`
		`alter table transactions ADD CONSTRAINT transaction_logs_block_number_fkey FOREIGN KEY (block_number) REFERENCES public.blocks(number) ON DELETE CASCADE;`

		`alter table blocks drop constraint block_sequences_pkey`,
		`alter table blocks add constraint block_pkey primary key (number)`,
		
		`alter column transaction_logs drop column block_sequence_id`,
		`alter column transactions drop column block_sequence_id`,
		`alter table blocks drop column id`,
	}

	for _, query := range queries {
		if _, err := db.Exec(query); err != nil {
			return err
		}
	}
	return nil
}

func (m *Migration) GetName() string {
	return "202409061121_removeSequenceId"
}
