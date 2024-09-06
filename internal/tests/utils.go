package tests

import (
	"github.com/Layr-Labs/sidecar/internal/config"
	"github.com/Layr-Labs/sidecar/internal/postgres"
	sqlite2 "github.com/Layr-Labs/sidecar/internal/sqlite"
	"gorm.io/gorm"
)

func GetConfig() *config.Config {
	return config.NewConfig()
}

func GetDatabaseConnection(cfg *config.Config) (*postgres.Postgres, *gorm.DB, error) {
	db, err := postgres.NewPostgres(&postgres.PostgresConfig{
		Host:     cfg.PostgresConfig.Host,
		Port:     cfg.PostgresConfig.Port,
		Username: cfg.PostgresConfig.Username,
		Password: cfg.PostgresConfig.Password,
		DbName:   cfg.PostgresConfig.DbName,
	})
	if err != nil {
		panic(err)
	}

	grm, err := postgres.NewGormFromPostgresConnection(db.Db)
	if err != nil {
		panic(err)
	}
	return db, grm, nil
}

const sqliteInMemoryPath = "file::memory:?cache=shared"

func GetSqliteDatabaseConnection() (*gorm.DB, error) {
	db, err := sqlite2.NewGormSqliteFromSqlite(sqlite2.NewSqlite(sqliteInMemoryPath))
	if err != nil {
		panic(err)
	}
	return db, nil
}
