package postgres

import (
	"database/sql"
	"fmt"
	"github.com/Layr-Labs/go-sidecar/internal/config"
	_ "github.com/lib/pq"
	"golang.org/x/xerrors"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type PostgresConfig struct {
	Host                string
	Port                int
	Username            string
	Password            string
	DbName              string
	CreateDbIfNotExists bool
}

type Postgres struct {
	Db *sql.DB
}

func PostgresConfigFromDbConfig(dbCfg *config.DatabaseConfig) *PostgresConfig {
	return &PostgresConfig{
		Host:     dbCfg.Host,
		Port:     dbCfg.Port,
		Username: dbCfg.User,
		Password: dbCfg.Password,
		DbName:   dbCfg.DbName,
	}
}

func CreateDatabaseIfNotExists(cfg *PostgresConfig) error {
	fmt.Printf("Creating database if not exists '%s'...\n", cfg.DbName)
	postgresConnStr := fmt.Sprintf("host=%s port=%d dbname=postgres sslmode=disable",
		cfg.Host,
		cfg.Port,
	)
	if cfg.Username != "" {
		postgresConnStr += fmt.Sprintf(" user=%s", cfg.Username)
	}
	if cfg.Password != "" {
		postgresConnStr += fmt.Sprintf(" password=%s", cfg.Password)
	}

	postgresDB, err := sql.Open("postgres", postgresConnStr)
	if err != nil {
		return fmt.Errorf("error connecting to postgres database: %v", err)
	}
	defer postgresDB.Close()

	// Check if database exists
	var exists bool
	query := fmt.Sprintf(`SELECT EXISTS(SELECT datname FROM pg_catalog.pg_database WHERE datname = '%s');`, cfg.DbName)
	err = postgresDB.QueryRow(query).Scan(&exists)
	if err != nil {
		return fmt.Errorf("error checking if database exists: %v", err)
	}

	// Create database if it doesn't exist
	if !exists {
		query = fmt.Sprintf("CREATE DATABASE %s", cfg.DbName)
		_, err = postgresDB.Exec(query)
		if err != nil {
			return fmt.Errorf("error creating database: %v", err)
		}
		fmt.Printf("Database '%s' created successfully\n", cfg.DbName)
	}
	return nil
}

func NewPostgres(cfg *PostgresConfig) (*Postgres, error) {
	if cfg.CreateDbIfNotExists {
		if err := CreateDatabaseIfNotExists(cfg); err != nil {
			return nil, xerrors.Errorf("Failed to create database if not exists", err)
		}
	}
	authString := ""
	if cfg.Username != "" {
		authString = fmt.Sprintf("%s user=%s", authString, cfg.Username)
	}
	if cfg.Password != "" {
		authString = fmt.Sprintf("%s password=%s", authString, cfg.Password)
	}
	connectString := fmt.Sprintf("host=%s %s dbname=%s port=%d sslmode=disable TimeZone=UTC",
		cfg.Host,
		authString,
		cfg.DbName,
		cfg.Port,
	)
	db, err := sql.Open("postgres", connectString)
	if err != nil {
		return nil, xerrors.Errorf("Failed to setup database", err)
	}

	return &Postgres{
		Db: db,
	}, nil
}

func NewGormFromPostgresConnection(pgDb *sql.DB) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.New(postgres.Config{
		Conn: pgDb,
	}), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, xerrors.Errorf("Failed to setup database", err)
	}

	return db, nil
}

func WrapTxAndCommit[T any](fn func(*gorm.DB) (T, error), db *gorm.DB, tx *gorm.DB) (T, error) {
	exists := tx != nil

	if !exists {
		tx = db.Begin()
	}

	res, err := fn(tx)

	if err != nil && !exists {
		fmt.Printf("Rollback transaction\n")
		tx.Rollback()
	}
	if err == nil && !exists {
		fmt.Printf("Commit transaction\n")
		tx.Commit()
	}
	return res, err
}
