package main

import (
	"github.com/Layr-Labs/sidecar/internal/config"
	"github.com/Layr-Labs/sidecar/internal/logger"
	"github.com/Layr-Labs/sidecar/internal/postgres"
	"github.com/Layr-Labs/sidecar/internal/postgres/migrations"
	"go.uber.org/zap"
	"log"
)

func main() {
	cfg := config.NewConfig()

	l, _ := logger.NewLogger(&logger.LoggerConfig{Debug: cfg.Debug})

	db, err := postgres.NewPostgres(&postgres.PostgresConfig{
		Host:     cfg.PostgresConfig.Host,
		Port:     cfg.PostgresConfig.Port,
		Username: cfg.PostgresConfig.Username,
		Password: cfg.PostgresConfig.Password,
		DbName:   cfg.PostgresConfig.DbName,
	})
	if err != nil {
		l.Error("Failed to setup postgres connection", zap.Error(err))
		panic(err)
	}

	grm, err := postgres.NewGormFromPostgresConnection(db.Db)
	if err != nil {
		l.Error("Failed to create gorm instance", zap.Error(err))
		panic(err)
	}

	migrator := migrations.NewMigrator(db.Db, grm, l)
	if err = migrator.MigrateAll(); err != nil {
		log.Fatalf("Failed to migrate: %v", err)
	}
}
