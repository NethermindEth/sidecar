package _202411061451_transactionLogsIndex

import (
	"database/sql"
	"gorm.io/gorm"
)

type Migration struct {
}

func (m *Migration) Up(db *sql.DB, grm *gorm.DB) error {
	query := `create index idx_transaction_logs_block_number on transaction_logs (block_number)`

	res := grm.Exec(query)
	if res.Error != nil {
		return res.Error
	}
	return nil
}

func (m *Migration) GetName() string {
	return "202411061451_transactionLogsIndex"
}
