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
	return gorm.Open(sqlite, &gorm.Config{})
}
