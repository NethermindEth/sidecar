package _202409111509_removeOperatorRestakedStrategiesBlockConstraint

import (
	"github.com/Layr-Labs/go-sidecar/internal/sqlite"
	"gorm.io/gorm"
)

type SqliteMigration struct {
}

func (m *SqliteMigration) Up(grm *gorm.DB) error {
	_, err := sqlite.WrapTxAndCommit[interface{}](func(_db *gorm.DB) (interface{}, error) {
		queries := []string{
			`PRAGMA foreign_keys = OFF;`,
			`CREATE TABLE IF NOT EXISTS new_operator_restaked_strategies (
				block_number INTEGER NOT NULL ,
				operator TEXT NOT NULL,
				avs TEXT NOT NULL,
				strategy TEXT NOT NULL,
				block_time DATETIME NOT NULL,
				avs_directory_address TEXT,
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
				updated_at DATETIME DEFAULT NULL,
				deleted_at DATETIME DEFAULT NULL
			);`,
			`INSERT INTO new_operator_restaked_strategies SELECT * FROM operator_restaked_strategies;`,
			`DROP TABLE operator_restaked_strategies;`,
			`ALTER TABLE new_operator_restaked_strategies RENAME TO operator_restaked_strategies;`,
			`PRAGMA foreign_keys = ON;`,
		}
		for _, query := range queries {
			res := grm.Exec(query)
			if res.Error != nil {
				return nil, res.Error
			}
		}
		return nil, nil
	}, grm, nil)
	return err
}

func (m *SqliteMigration) GetName() string {
	return "202409111509_removeOperatorRestakedStrategiesBlockConstraint"
}
