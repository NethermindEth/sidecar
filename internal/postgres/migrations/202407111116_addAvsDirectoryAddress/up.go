package _202407111116_addAvsDirectoryAddress

import (
	"database/sql"
	"fmt"
	"github.com/Layr-Labs/sidecar/internal/postgres"
	"gorm.io/gorm"
)

type Migration struct {
}

func (m *Migration) Up(db *sql.DB, grm *gorm.DB) error {
	queries := []string{
		`ALTER TABLE operator_restaked_strategies add column avs_directory_address varchar`,
	}

	_, err := postgres.WrapTxAndCommit[interface{}](func(tx *gorm.DB) (interface{}, error) {
		for _, query := range queries {
			err := tx.Exec(query)
			if err.Error != nil {
				fmt.Printf("Failed to run migration query: %s - %+v\n", query, err.Error)
				return 0, err.Error
			}
		}
		return 0, nil
	}, nil, grm)
	return err
}

func (m *Migration) GetName() string {
	return "202407111116_addAvsDirectoryAddress"
}
