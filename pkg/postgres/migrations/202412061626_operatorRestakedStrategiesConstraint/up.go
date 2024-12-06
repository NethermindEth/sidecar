package _202412061626_operatorRestakedStrategiesConstraint

import (
	"database/sql"
	"github.com/Layr-Labs/sidecar/internal/config"
	"gorm.io/gorm"
)

type Migration struct {
}

func (m *Migration) Up(db *sql.DB, grm *gorm.DB, cfg *config.Config) error {
	query := `
		alter table operator_restaked_strategies add constraint uniq_operator_restaked_strategies unique (block_number, operator, avs, strategy)
	`
	res := grm.Exec(query)
	if res.Error != nil {
		return res.Error
	}
	return nil
}

func (m *Migration) GetName() string {
	return "202412061626_operatorRestakedStrategiesConstraint"
}
