package _202411081039_operatorRestakedStrategiesConstraint

import (
	"database/sql"
	"github.com/Layr-Labs/sidecar/internal/config"
	"gorm.io/gorm"
)

type Migration struct {
}

func (m *Migration) Up(db *sql.DB, grm *gorm.DB, cfg *config.Config) error {
	query := `
		create unique index idx_unique_operator_restaked_strategies on operator_restaked_strategies(block_number, operator, avs, strategy)
	`
	res := grm.Exec(query)
	if res.Error != nil {
		return res.Error
	}
	return nil
}

func (m *Migration) GetName() string {
	return "202411081039_operatorRestakedStrategiesConstraint"
}
