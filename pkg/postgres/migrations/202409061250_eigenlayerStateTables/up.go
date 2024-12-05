package _202409061250_eigenlayerStateTables

import (
	"database/sql"
	"fmt"
	"github.com/Layr-Labs/sidecar/internal/config"

	"gorm.io/gorm"
)

type Migration struct {
}

func (m *Migration) Up(db *sql.DB, grm *gorm.DB, cfg *config.Config) error {
	queries := []string{
		`create table if not exists registered_avs_operators (
			operator varchar not null,
			avs varchar not null,
			block_number bigint not null,
			created_at timestamp with time zone DEFAULT current_timestamp,
			unique(operator, avs, block_number)
		);
		`,
	}

	for _, query := range queries {
		if res := grm.Exec(query); res.Error != nil {
			fmt.Printf("Failed to execute query: %s\n", query)
			return res.Error
		}
	}
	return nil
}

func (m *Migration) GetName() string {
	return "202409061250_eigenlayerStateTables"
}
