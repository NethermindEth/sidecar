package sqlite

import (
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func NewSqlite(path string) gorm.Dialector {
	db := sqlite.Open(path)
	return db
}

func NewGormSqliteFromSqlite(sqlite gorm.Dialector) (*gorm.DB, error) {
	db, err := gorm.Open(sqlite, &gorm.Config{})
	if err != nil {
		return nil, err
	}

	pragmas := []string{
		`PRAGMA foreign_keys = ON;`,
		`PRAGMA journal_mode = WAL;`,
	}

	for _, pragma := range pragmas {
		res := db.Exec(pragma)
		if res.Error != nil {
			return nil, res.Error
		}
	}
	return db, nil
}
