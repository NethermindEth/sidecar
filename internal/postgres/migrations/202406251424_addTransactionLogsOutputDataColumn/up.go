package _202406251424_addTransactionLogsOutputDataColumn

import (
	"database/sql"
	"gorm.io/gorm"
)

type Migration struct {
}

func (m *Migration) Up(db *sql.DB, grm *gorm.DB) error {
	query := `alter table transaction_logs add column output_data jsonb;`

	_, err := db.Exec(query)
	if err != nil {
		return err
	}
	return nil
}

func (m *Migration) GetName() string {
	return "202406251424_addTransactionLogsOutputDataColumn"
}
