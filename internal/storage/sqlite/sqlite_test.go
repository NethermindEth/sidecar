package sqlite

import (
	"fmt"
	"github.com/Layr-Labs/sidecar/internal/config"
	"github.com/Layr-Labs/sidecar/internal/logger"
	"github.com/Layr-Labs/sidecar/internal/storage"
	"github.com/Layr-Labs/sidecar/internal/storage/sqlite/migrations"
	"github.com/Layr-Labs/sidecar/internal/tests"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"testing"
	"time"
)

func setup() (*gorm.DB, *zap.Logger, *config.Config) {
	cfg := config.NewConfig()
	l, err := logger.NewLogger(&logger.LoggerConfig{Debug: true})
	db, err := tests.GetSqliteDatabaseConnection()
	if err != nil {
		panic(err)
	}
	sqliteMigrator := migrations.NewSqliteMigrator(db, l)
	if err := sqliteMigrator.MigrateAll(); err != nil {
		l.Sugar().Fatalw("Failed to migrate", "error", err)
	}
	return db, l, cfg
}

func teardown(db *gorm.DB, l *zap.Logger) {
	queries := []string{
		`truncate table blocks cascade`,
		`truncate table transactions cascade`,
		`truncate table transaction_logs cascade`,
		`truncate table transaction_logs cascade`,
	}
	for _, query := range queries {
		res := db.Exec(query)
		if res.Error != nil {
			l.Sugar().Errorw("Failed to truncate table", "error", res.Error)
		}
	}
}

func Test_SqliteBlockstore(t *testing.T) {
	t.Run("Blocks", func(t *testing.T) {
		db, l, cfg := setup()

		sqliteStore := NewSqliteBlockStore(db, l, cfg)

		t.Run("InsertBlockAtHeight", func(t *testing.T) {
			block := &storage.Block{
				Number:    100,
				Hash:      "some hash",
				BlockTime: time.Now(),
			}

			insertedBlock, err := sqliteStore.InsertBlockAtHeight(block.Number, block.Hash, uint64(block.BlockTime.Unix()))
			if err != nil {
				t.Errorf("Failed to insert block: %v", err)
			}
			assert.NotNil(t, insertedBlock)
			assert.Equal(t, block.Number, insertedBlock.Number)
			assert.Equal(t, block.Hash, insertedBlock.Hash)
		})
		t.Run("Fail to insert a duplicate block", func(t *testing.T) {
			block := &storage.Block{
				Number:    100,
				Hash:      "some hash",
				BlockTime: time.Now(),
			}

			_, err := sqliteStore.InsertBlockAtHeight(block.Number, block.Hash, uint64(block.BlockTime.Unix()))
			assert.NotNil(t, err)
			assert.Contains(t, err.Error(), "UNIQUE constraint failed")
			fmt.Printf("Error: %v\n", err)
		})
		t.Run("InsertBlockTransaction", func(t *testing.T) {

		})
	})
}
