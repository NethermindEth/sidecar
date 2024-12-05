package _202409181340_stakerDelegationDelta

import (
	"database/sql"
	"github.com/Layr-Labs/sidecar/internal/config"
	"gorm.io/gorm"
)

type Migration struct {
}

func (m *Migration) Up(db *sql.DB, grm *gorm.DB, cfg *config.Config) error {
	query := `
		create table if not exists staker_delegation_changes (
			staker varchar not null,
			operator varchar not null,
			delegated boolean not null,
			block_number bigint not null,
			log_index integer not null
		)
	`
	res := grm.Exec(query)
	if res.Error != nil {
		return res.Error
	}
	return nil
}

func (m *Migration) GetName() string {
	return "202409181340_stakerDelegationDelta"
}
