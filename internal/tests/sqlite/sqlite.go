package sqlite

import (
	sqlite2 "github.com/Layr-Labs/go-sidecar/internal/sqlite"
	"github.com/Layr-Labs/go-sidecar/internal/tests"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func GetInMemorySqliteDatabaseConnection(l *zap.Logger) (*gorm.DB, error) {
	db, err := sqlite2.NewGormSqliteFromSqlite(sqlite2.NewSqlite(&sqlite2.SqliteConfig{
		Path:           sqlite2.SqliteInMemoryPath,
		ExtensionsPath: []string{tests.GetSqliteExtensionsPath()},
	}, l))
	if err != nil {
		panic(err)
	}
	return db, nil
}
