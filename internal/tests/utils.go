package tests

import (
	"github.com/Layr-Labs/sidecar/internal/config"
	sqlite2 "github.com/Layr-Labs/sidecar/internal/sqlite"
	"gorm.io/gorm"
)

func GetConfig() *config.Config {
	return config.NewConfig()
}

const sqliteInMemoryPath = "file::memory:?cache=shared"

func GetSqliteDatabaseConnection() (*gorm.DB, error) {
	db, err := sqlite2.NewGormSqliteFromSqlite(sqlite2.NewSqlite(sqliteInMemoryPath))
	if err != nil {
		panic(err)
	}
	return db, nil
}
