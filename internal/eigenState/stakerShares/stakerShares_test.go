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
	"strings"
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
		assert.Equal(t, "0xaf6fb48ac4a60c61a64124ce9dc28f508dc8de8d", typedChange.Staker)
		assert.Equal(t, "0x7d704507b76571a51d9cae8addabbfd0ba0e63d3", typedChange.Strategy)

		teardown(model)
	})
	t.Run("Should capture a staker share M1 Withdrawal", func(t *testing.T) {
		esm := stateManager.NewEigenStateManager(l, grm)
		blockNumber := uint64(200)
		log := storage.TransactionLog{
			TransactionHash:  "some hash",
			TransactionIndex: big.NewInt(200).Uint64(),
			BlockNumber:      blockNumber,
			Address:          cfg.GetContractsMapForEnvAndNetwork().StrategyManager,
			Arguments:        `[{"Name": "depositor", "Type": "address", "Value": null, "Indexed": false}, {"Name": "nonce", "Type": "uint96", "Value": null, "Indexed": false}, {"Name": "strategy", "Type": "address", "Value": null, "Indexed": false}, {"Name": "shares", "Type": "uint256", "Value": null, "Indexed": false}]`,
			EventName:        "ShareWithdrawalQueued",
			LogIndex:         big.NewInt(500).Uint64(),
			OutputData:       `{"nonce": 0, "shares": 246393621132195985, "strategy": "0x298afb19a105d59e74658c4c334ff360bade6dd2", "depositor": "0x9c01148c464cf06d135ad35d3d633ab4b46b9b78"}`,
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

		expectedShares, _ := uint256.FromDecimal("246393621132195985")
		assert.Equal(t, expectedShares, typedChange.Shares)
		assert.Equal(t, "0x9c01148c464cf06d135ad35d3d633ab4b46b9b78", typedChange.Staker)
		assert.Equal(t, "0x298afb19a105d59e74658c4c334ff360bade6dd2", typedChange.Strategy)

		teardown(model)
	})
	t.Run("Should capture staker EigenPod shares", func(t *testing.T) {
		esm := stateManager.NewEigenStateManager(l, grm)
		blockNumber := uint64(200)
		log := storage.TransactionLog{
			TransactionHash:  "some hash",
			TransactionIndex: big.NewInt(300).Uint64(),
			BlockNumber:      blockNumber,
			Address:          cfg.GetContractsMapForEnvAndNetwork().EigenpodManager,
			Arguments:        `[{"Name": "podOwner", "Type": "address", "Value": "0x0808D4689B347D499a96f139A5fC5B5101258406"}, {"Name": "sharesDelta", "Type": "int256", "Value": ""}]`,
			EventName:        "PodSharesUpdated",
			LogIndex:         big.NewInt(600).Uint64(),
			OutputData:       `{"sharesDelta": 32000000000000000000}`,
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

		expectedShares, _ := uint256.FromDecimal("32000000000000000000")
		assert.Equal(t, expectedShares, typedChange.Shares)
		assert.Equal(t, strings.ToLower("0x0808D4689B347D499a96f139A5fC5B5101258406"), typedChange.Staker)
		assert.Equal(t, "0xbeac0eeeeeeeeeeeeeeeeeeeeeeeeeeeeeebeac0", typedChange.Strategy)

		teardown(model)
	})
	t.Run("Should capture M2 migrated withdrawals", func(t *testing.T) {
		t.Skip("M2 migration is not yet implemented")
	})
}
