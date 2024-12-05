package _202409161057_avsOperatorDeltas

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
		`create table if not exists avs_operator_state_changes (
			operator varchar not null,
			avs varchar not null,
			block_number bigint not null,
			log_index integer not null,
			registered boolean not null,
			created_at timestamp with time zone DEFAULT current_timestamp,
			unique(operator, avs, block_number, log_index)
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
	return "202409161057_avsOperatorDeltas"
}
