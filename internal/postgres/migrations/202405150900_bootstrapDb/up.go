package _202405150900_bootstrapDb

import (
	"database/sql"
	"fmt"
	"github.com/Layr-Labs/sidecar/internal/postgres"
	"gorm.io/gorm"
)

type Migration struct {
}

func (m *Migration) Up(db *sql.DB, grm *gorm.DB) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS block_sequences (
			id serial PRIMARY KEY,
			number bigint NOT NULL,
			hash varchar(255) NOT NULL,
			blob_path text NOT NULL,
			created_at timestamp with time zone DEFAULT current_timestamp,
			updated_at timestamp with time zone DEFAULT NULL,
			deleted_at timestamp with time zone DEFAULT NULL
    	)`,
		`CREATE TABLE IF NOT EXISTS transactions (
			sequence_id bigint NOT NULL REFERENCES block_sequences(id) ON DELETE CASCADE,
			block_number bigint NOT NULL,
			transaction_hash varchar(255) NOT NULL,
			transaction_index bigint NOT NULL,
			created_at timestamp with time zone DEFAULT current_timestamp,
			updated_at timestamp with time zone DEFAULT NULL,
			deleted_at timestamp with time zone DEFAULT NULL,
			UNIQUE(transaction_hash)
		)`,
		`CREATE UNIQUE INDEX idx_sequence_id_tx_hash_tx_index on transactions(sequence_id, transaction_hash, transaction_index)`,
		`CREATE TABLE IF NOT EXISTS transaction_logs (
			transaction_hash varchar(255) NOT NULL REFERENCES transactions(transaction_hash) ON DELETE CASCADE,
			address varchar(255) NOT NULL,
			arguments JSONB DEFAULT NULL,
			event_name varchar(255) NOT NULL,
			log_index bigint NOT NULL,
			created_at timestamp with time zone DEFAULT current_timestamp,
			updated_at timestamp with time zone DEFAULT NULL,
			deleted_at timestamp with time zone DEFAULT NULL
    	)`,
		`CREATE UNIQUE INDEX idx_transaction_hash on transaction_logs(transaction_hash, log_index)`,
		`CREATE TABLE IF NOT EXISTS contracts (
			contract_address varchar(255) PRIMARY KEY,
			contract_abi text NOT NULL,
			created_at timestamp with time zone DEFAULT current_timestamp,
			updated_at timestamp with time zone DEFAULT NULL,
			deleted_at timestamp with time zone DEFAULT NULL,
			UNIQUE(contract_address)
		)`,
	}

	_, err := postgres.WrapTxAndCommit[interface{}](func(tx *gorm.DB) (interface{}, error) {
		for _, query := range queries {
			err := tx.Exec(query)
			if err.Error != nil {
				fmt.Printf("Failed to run migration query: %s - %+v\n", query, err.Error)
				return 0, err.Error
			}
		}
		return 0, nil
	}, grm, nil)
	return err
}

func (m *Migration) GetName() string {
	return "202405150900_bootstrapDb"
}
