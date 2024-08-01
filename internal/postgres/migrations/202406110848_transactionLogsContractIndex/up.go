package _202406110848_transactionLogsContractIndex

import (
	"database/sql"
	"gorm.io/gorm"
)

type Migration struct {
}

func (m *Migration) Up(db *sql.DB, grm *gorm.DB) error {
	query := `create index concurrently idx_transaction_logs_address on transaction_logs(address)`

	_, err := db.Exec(query)
	if err != nil {
		return err
	}
	return nil
}

func (m *Migration) GetName() string {
	return "202406110848_transactionLogsContractIndex"
}
