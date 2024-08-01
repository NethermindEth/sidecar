package postgres

import (
	"database/sql"
	"fmt"
	_ "github.com/lib/pq"
	"golang.org/x/xerrors"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type PostgresConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	DbName   string
}

type Postgres struct {
	Db *sql.DB
}

func NewPostgres(cfg *PostgresConfig) (*Postgres, error) {
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
