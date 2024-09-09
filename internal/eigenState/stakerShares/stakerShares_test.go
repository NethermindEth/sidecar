package stakerShares

import (
	"github.com/Layr-Labs/sidecar/internal/config"
	"github.com/Layr-Labs/sidecar/internal/eigenState/stateManager"
	"github.com/Layr-Labs/sidecar/internal/logger"
	"github.com/Layr-Labs/sidecar/internal/sqlite/migrations"
	"github.com/Layr-Labs/sidecar/internal/storage"
	"github.com/Layr-Labs/sidecar/internal/tests"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"math/big"
	"testing"
	"time"
)

func setup() (
	*config.Config,
	*gorm.DB,
	*zap.Logger,
	error,
) {
	cfg := tests.GetConfig()
	l, _ := logger.NewLogger(&logger.LoggerConfig{Debug: cfg.Debug})

	db, err := tests.GetSqliteDatabaseConnection()
	if err != nil {
		panic(err)
	}
	sqliteMigrator := migrations.NewSqliteMigrator(db, l)
	if err := sqliteMigrator.MigrateAll(); err != nil {
		l.Sugar().Fatalw("Failed to migrate", "error", err)
	}

	return cfg, db, l, err
}

func teardown(model *StakerSharesModel) {
	model.Db.Exec("truncate table staker_shares cascade")
}

func Test_StakerSharesState(t *testing.T) {
	cfg, grm, l, err := setup()

	if err != nil {
		t.Fatal(err)
	}

	t.Run("Should create a new OperatorSharesState", func(t *testing.T) {
		esm := stateManager.NewEigenStateManager(l, grm)
		model, err := NewStakerSharesModel(esm, grm, cfg.Network, cfg.Environment, l, cfg)
		assert.Nil(t, err)
		assert.NotNil(t, model)
	})
	t.Run("Should capture a staker share Deposit", func(t *testing.T) {
		esm := stateManager.NewEigenStateManager(l, grm)
		blockNumber := uint64(200)
		log := storage.TransactionLog{
			TransactionHash:  "some hash",
			TransactionIndex: big.NewInt(100).Uint64(),
			BlockNumber:      blockNumber,
			Address:          cfg.GetContractsMapForEnvAndNetwork().StrategyManager,
			Arguments:        `[{"Name": "staker", "Type": "address", "Value": ""}, {"Name": "token", "Type": "address", "Value": ""}, {"Name": "strategy", "Type": "address", "Value": ""}, {"Name": "shares", "Type": "uint256", "Value": ""}]`,
			EventName:        "Deposit",
			LogIndex:         big.NewInt(400).Uint64(),
			OutputData:       `{"token": "0x3f1c547b21f65e10480de3ad8e19faac46c95034", "shares": 159925690037480381, "staker": "0xaf6fb48ac4a60c61a64124ce9dc28f508dc8de8d", "strategy": "0x7d704507b76571a51d9cae8addabbfd0ba0e63d3"}`,
			CreatedAt:        time.Time{},
			UpdatedAt:        time.Time{},
			DeletedAt:        time.Time{},
		}

		model, err := NewStakerSharesModel(esm, grm, cfg.Network, cfg.Environment, l, cfg)

		err = model.InitBlockProcessing(blockNumber)
		assert.Nil(t, err)

		change, err := model.HandleStateChange(&log)
		assert.Nil(t, err)
		assert.NotNil(t, change)

		typedChange := change.(*AccumulatedStateChange)

		expectedShares, _ := uint256.FromDecimal("159925690037480381")
		assert.Equal(t, expectedShares, typedChange.Shares)

		teardown(model)
	})
}
