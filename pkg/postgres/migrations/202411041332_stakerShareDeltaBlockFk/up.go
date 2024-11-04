package _202411041332_stakerShareDeltaBlockFk

import (
	"database/sql"
	"gorm.io/gorm"
)

type Migration struct {
}

func (m *Migration) Up(db *sql.DB, grm *gorm.DB) error {
	queries := []string{
		`alter table staker_share_deltas add constraint staker_share_deltas_block_number_fkey foreign key (block_number) references blocks (number) on delete cascade`,
		`alter table avs_operator_state_changes add constraint avs_operator_state_changes_block_number_fkey foreign key (block_number) references blocks (number) on delete cascade`,
	}
	for _, query := range queries {
		_, err := db.Exec(query)
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *Migration) GetName() string {
	return "202411041332_stakerShareDeltaBlockFk"
}
