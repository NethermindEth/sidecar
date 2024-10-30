package postgres

import (
	"github.com/Layr-Labs/go-sidecar/internal/config"
	"github.com/Layr-Labs/go-sidecar/internal/logger"
	"github.com/Layr-Labs/go-sidecar/internal/tests"
	"github.com/Layr-Labs/go-sidecar/pkg/postgres/migrations"
	"testing"
)

func Test_Postgres(t *testing.T) {
	cfg := config.NewConfig()
	cfg.DatabaseConfig = *tests.GetDbConfigFromEnv()

	testDbName, err := tests.GenerateTestDbName()
	if err != nil {
		t.Fatalf("Failed to generate test database name: %v", err)
	}
	cfg.DatabaseConfig.DbName = testDbName

	l, _ := logger.NewLogger(&logger.LoggerConfig{Debug: true})

	pgConfig := PostgresConfigFromDbConfig(&cfg.DatabaseConfig)
	pgConfig.CreateDbIfNotExists = true
	pg, err := NewPostgres(pgConfig)
	if err != nil {
		t.Fatalf("Failed to setup postgres: %v", err)
	}

	grm, err := NewGormFromPostgresConnection(pg.Db)
	if err != nil {
		t.Fatalf("Failed to create gorm instance: %v", err)
	}

	t.Run("Test Migration Up", func(t *testing.T) {
		migrator := migrations.NewMigrator(pg.Db, grm, l)
		if err = migrator.MigrateAll(); err != nil {
			t.Fatalf("Failed to migrate: %v", err)
		}
	})
}
