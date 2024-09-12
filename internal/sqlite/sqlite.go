package sqlite

import (
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/Layr-Labs/go-sidecar/internal/types/numbers"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"os"
	"path/filepath"
	"regexp"

	goSqlite "github.com/mattn/go-sqlite3"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// bytesToHex is a custom SQLite function that converts a JSON byte array to a hex string.
//
// @param jsonByteArray: a JSON byte array, e.g. [1, 2, 3, ...]
// @return: a hex string without a leading 0x, e.g. 78cc56f0700e7ba5055f12...
func bytesToHex(jsonByteArray string) (string, error) {
	jsonBytes := make([]byte, 0)
	err := json.Unmarshal([]byte(jsonByteArray), &jsonBytes)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(jsonBytes), nil
}

func InitSqliteDir(path string) error {
	// strip the db file from the path and check if the directory already exists
	basePath := filepath.Dir(path)

	// check if the directory already exists
	if _, err := os.Stat(basePath); os.IsNotExist(err) {
		// create the directory
		if err := os.MkdirAll(basePath, os.ModePerm); err != nil {
			return err
		}
	}
	return nil
}

type SumBigNumbers struct {
	total decimal.Decimal
}

func NewSumBigNumbers() *SumBigNumbers {
	zero, _ := decimal.NewFromString("0")
	return &SumBigNumbers{total: zero}
}

func (s *SumBigNumbers) Step(value any) {
	bigValue, err := decimal.NewFromString(value.(string))
	if err != nil {
		return
	}
	s.total = s.total.Add(bigValue)
}

func (s *SumBigNumbers) Done() (string, error) {
	return s.total.String(), nil
}

var hasRegisteredExtensions = false

const SqliteInMemoryPath = "file::memory:?cache=shared"

func NewInMemorySqlite(l *zap.Logger) gorm.Dialector {
	return NewSqlite(SqliteInMemoryPath, l)
}

func NewInMemorySqliteWithName(name string, l *zap.Logger) gorm.Dialector {
	path := fmt.Sprintf("file:%s?mode=memory&cache=shared", name)
	return NewSqlite(path, l)
}

func NewSqlite(path string, l *zap.Logger) gorm.Dialector {
	if !hasRegisteredExtensions {
		sql.Register("sqlite3_with_extensions", &goSqlite.SQLiteDriver{
			ConnectHook: func(conn *goSqlite.SQLiteConn) error {
				// Generic functions
				if err := conn.RegisterAggregator("sum_big", NewSumBigNumbers, true); err != nil {
					l.Sugar().Errorw("Failed to register aggregator sum_big", "error", err)
					return err
				}
				if err := conn.RegisterFunc("subtract_big", numbers.SubtractBig, true); err != nil {
					l.Sugar().Errorw("Failed to register function subtract_big", "error", err)
					return err
				}
				if err := conn.RegisterFunc("numeric_multiply", numbers.NumericMultiply, true); err != nil {
					l.Sugar().Errorw("Failed to register function NumericMultiply", "error", err)
					return err
				}
				if err := conn.RegisterFunc("big_gt", numbers.BigGreaterThan, true); err != nil {
					l.Sugar().Errorw("Failed to register function BigGreaterThan", "error", err)
					return err
				}

				if err := conn.RegisterFunc("bytes_to_hex", bytesToHex, true); err != nil {
					l.Sugar().Errorw("Failed to register function bytes_to_hex", "error", err)
					return err
				}
				// Raw tokens per day
				if err := conn.RegisterFunc("calc_raw_tokens_per_day", numbers.CalcRawTokensPerDay, true); err != nil {
					l.Sugar().Errorw("Failed to register function calc_raw_tokens_per_day", "error", err)
					return err
				}
				// Forked tokens per day
				if err := conn.RegisterFunc("post_nile_tokens_per_day", numbers.PostNileTokensPerDay, true); err != nil {
					l.Sugar().Errorw("Failed to register function PostNileTokensPerDay", "error", err)
					return err
				}
				if err := conn.RegisterFunc("pre_nile_tokens_per_day", numbers.PreNileTokensPerDay, true); err != nil {
					l.Sugar().Errorw("Failed to register function PreNileTokensPerDay", "error", err)
					return err
				}
				// Staker proportion and weight
				if err := conn.RegisterFunc("calc_staker_proportion", numbers.CalculateStakerProportion, true); err != nil {
					l.Sugar().Errorw("Failed to register function CalculateStakerProportion", "error", err)
					return err
				}
				if err := conn.RegisterFunc("calc_staker_weight", numbers.CalculateStakerWeight, true); err != nil {
					l.Sugar().Errorw("Failed to register function CalculateStakerWeight", "error", err)
					return err
				}
				// Forked rewards for stakers
				if err := conn.RegisterFunc("amazon_token_rewards", numbers.CalculateAmazonStakerTokenRewards, true); err != nil {
					l.Sugar().Errorw("Failed to register function CalculateAmazonStakerTokenRewards", "error", err)
					return err
				}
				if err := conn.RegisterFunc("nile_token_rewards", numbers.CalculateNileStakerTokenRewards, true); err != nil {
					l.Sugar().Errorw("Failed to register function CalculateNileStakerTokenRewards", "error", err)
					return err
				}
				if err := conn.RegisterFunc("post_nile_token_rewards", numbers.CalculatePostNileStakerTokenRewards, true); err != nil {
					l.Sugar().Errorw("Failed to register function CalculatePostNileStakerTokenRewards", "error", err)
					return err
				}
				// Operator tokens
				if err := conn.RegisterFunc("amazon_operator_tokens", numbers.CalculateAmazonOperatorTokens, true); err != nil {
					l.Sugar().Errorw("Failed to register function CalculateAmazonOperatorTokens", "error", err)
					return err
				}
				if err := conn.RegisterFunc("nile_operator_tokens", numbers.CalculateNileOperatorTokens, true); err != nil {
					l.Sugar().Errorw("Failed to register function CalculateNileOperatorTokens", "error", err)
					return err
				}
				if err := conn.RegisterFunc("post_nile_operator_tokens", numbers.CalculatePostNileOperatorTokens, true); err != nil {
					l.Sugar().Errorw("Failed to register function CalculatePostNileOperatorTokens", "error", err)
					return err
				}
				return nil
			},
		})
		hasRegisteredExtensions = true
	}

	return &sqlite.Dialector{
		DriverName: "sqlite3_with_extensions",
		DSN:        path,
	}
}

func NewGormSqliteFromSqlite(sqlite gorm.Dialector) (*gorm.DB, error) {
	db, err := gorm.Open(sqlite, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, err
	}

	// https://phiresky.github.io/blog/2020/sqlite-performance-tuning/
	pragmas := []string{
		`PRAGMA foreign_keys = ON;`,
		`PRAGMA journal_mode = WAL;`,
		`PRAGMA synchronous = normal;`,
		`pragma mmap_size = 30000000000;`,
	}

	for _, pragma := range pragmas {
		res := db.Exec(pragma)
		if res.Error != nil {
			return nil, res.Error
		}
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
		tx.Commit()
	}
	return res, err
}

func IsDuplicateKeyError(err error) bool {
	r := regexp.MustCompile(`UNIQUE constraint failed`)

	return r.MatchString(err.Error())
}
