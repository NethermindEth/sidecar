package _202409061121_removeSequenceId

import (
	"database/sql"
	"fmt"
	"gorm.io/gorm"
)

type Migration struct {
}

func (m *Migration) Up(db *sql.DB, grm *gorm.DB) error {
	queries := []string{
		`alter table blocks add constraint unique_blocks unique (number)`,

		`alter table transactions add constraint transactions_transaction_hash_block_number_key unique (transaction_hash, block_number)`,
		`alter table transactions ADD CONSTRAINT transactions_block_number_fkey FOREIGN KEY (block_number) REFERENCES blocks("number") ON DELETE CASCADE;`,
		`alter table transaction_logs ADD CONSTRAINT fk_transaction_hash_block_number_key FOREIGN KEY (transaction_hash, block_number) REFERENCES public.transactions(transaction_hash, block_number) ON DELETE CASCADE;`,
		`alter table transaction_logs ADD CONSTRAINT transaction_logs_block_number_fkey FOREIGN KEY (block_number) REFERENCES public.blocks("number") ON DELETE CASCADE;`,

		// drop constraints
		`alter table transaction_logs drop constraint fk_transaction_hash_sequence_id_key`,
		`alter table transaction_logs drop constraint transaction_logs_block_sequence_id_fkey`,

		`alter table transactions drop constraint transactions_transaction_hash_sequence_id_key`,
		`alter table transactions drop constraint transactions_block_sequence_id_fkey`,

		`alter table transactions drop column block_sequence_id`,
		`alter table transaction_logs drop column block_sequence_id`,
		// re-add constraints

		`alter table blocks drop constraint block_sequences_pkey`,
		`alter table blocks add constraint block_pkey primary key (number)`,

		`alter table blocks drop column id`,
	}

	for _, query := range queries {
		if _, err := db.Exec(query); err != nil {
			fmt.Println("Error executing query: ", query)
			return err
		}
	}
	return nil
}

func (m *Migration) GetName() string {
	return "202409061121_removeSequenceId"
}
