package _202405170842_addBlockInfoToTransactionLog

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
		`ALTER TABLE transaction_logs ADD COLUMN block_number bigint NOT NULL, ADD COLUMN transaction_index integer NOT NULL`,
		`ALTER TABLE transaction_logs ADD COLUMN block_sequence_id bigint NOT NULL references block_sequences(id) on delete cascade`,
		`ALTER TABLE transactions RENAME COLUMN sequence_id TO block_sequence_id`,
		`ALTER TABLE block_sequences RENAME TO blocks`,
		`ALTER TABLE blocks add column block_time timestamp with time zone NOT NULL`,
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
	}, nil, grm)
	return err
}

func (m *Migration) GetName() string {
	return "202405170842_addBlockInfoToTransactionLog"
}
