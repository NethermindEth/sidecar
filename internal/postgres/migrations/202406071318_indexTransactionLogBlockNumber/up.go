package _202406071318_indexTransactionLogBlockNumber

import (
	"database/sql"
	"gorm.io/gorm"
)

type Migration struct {
}

func (m *Migration) Up(db *sql.DB, grm *gorm.DB) error {
	query := `create index concurrently if not exists idx_transaciton_logs_block_number on transaction_logs (block_number);`

	_, err := db.Exec(query)
	if err != nil {
		return err
	}
	return nil
}

func (m *Migration) GetName() string {
	return "202406071318_indexTransactionLogBlockNumber"
}
