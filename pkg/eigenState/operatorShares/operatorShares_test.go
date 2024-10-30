package operatorShares

import (
	"database/sql"
	"github.com/Layr-Labs/go-sidecar/pkg/postgres"
	"github.com/Layr-Labs/go-sidecar/pkg/storage"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/Layr-Labs/go-sidecar/internal/config"
	"github.com/Layr-Labs/go-sidecar/internal/logger"
	"github.com/Layr-Labs/go-sidecar/internal/tests"
	"github.com/Layr-Labs/go-sidecar/pkg/eigenState/stateManager"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func setup() (
	string,
	*gorm.DB,
	*zap.Logger,
	*config.Config,
	error,
) {
	cfg := config.NewConfig()
	cfg.DatabaseConfig = *tests.GetDbConfigFromEnv()

	l, _ := logger.NewLogger(&logger.LoggerConfig{Debug: true})

	dbname, _, grm, err := postgres.GetTestPostgresDatabase(cfg.DatabaseConfig, l)
	if err != nil {
		return dbname, nil, nil, nil, err
	}

	return dbname, grm, l, cfg, nil
}

func teardown(model *OperatorSharesModel) {
	model.DB.Exec("truncate table operator_shares")
}

func createBlock(model *OperatorSharesModel, blockNumber uint64) error {
	block := &storage.Block{
		Number:    blockNumber,
		Hash:      "some hash",
		BlockTime: time.Now().Add(time.Hour * time.Duration(blockNumber)),
	}
	res := model.DB.Model(&storage.Block{}).Create(block)
	if res.Error != nil {
		return res.Error
	}
	return nil
}

func Test_OperatorSharesState(t *testing.T) {
	dbName, grm, l, cfg, err := setup()

	if err != nil {
		t.Fatal(err)
	}

	t.Run("Should create a new OperatorSharesState", func(t *testing.T) {
		esm := stateManager.NewEigenStateManager(l, grm)
		model, err := NewOperatorSharesModel(esm, grm, l, cfg)
		assert.Nil(t, err)
		assert.NotNil(t, model)
	})
	t.Run("Should register OperatorSharesState", func(t *testing.T) {
		esm := stateManager.NewEigenStateManager(l, grm)
		blockNumber := uint64(200)
		log := storage.TransactionLog{
			TransactionHash:  "some hash",
			TransactionIndex: big.NewInt(100).Uint64(),
			BlockNumber:      blockNumber,
			Address:          cfg.GetContractsMapForChain().DelegationManager,
			Arguments:        `[{"Value": "0xdb9afbdcfeca94dfb25790c900c527969e78bd3c"}]`,
			EventName:        "OperatorSharesIncreased",
			LogIndex:         big.NewInt(400).Uint64(),
			OutputData:       `{"shares": "100", "strategy": "0x93c4b944d05dfe6df7645a86cd2206016c51564d"}`,
			CreatedAt:        time.Time{},
			UpdatedAt:        time.Time{},
			DeletedAt:        time.Time{},
		}

		model, err := NewOperatorSharesModel(esm, grm, l, cfg)
		assert.Nil(t, err)

		err = model.SetupStateForBlock(blockNumber)
		assert.Nil(t, err)

		change, err := model.HandleStateChange(&log)
		assert.Nil(t, err)
		assert.NotNil(t, change)

		t.Cleanup(func() {
			teardown(model)
		})
	})
	t.Run("Should register AvsOperatorState and generate the table for the block", func(t *testing.T) {
		esm := stateManager.NewEigenStateManager(l, grm)
		blockNumber := uint64(200)

		model, err := NewOperatorSharesModel(esm, grm, l, cfg)
		assert.Nil(t, err)

		err = model.SetupStateForBlock(blockNumber)
		assert.Nil(t, err)

		err = createBlock(model, blockNumber)
		assert.Nil(t, err)

		log := storage.TransactionLog{
			TransactionHash:  "some hash",
			TransactionIndex: big.NewInt(100).Uint64(),
			BlockNumber:      blockNumber,
			Address:          cfg.GetContractsMapForChain().DelegationManager,
			Arguments:        `[{"Value": "0xdb9afbdcfeca94dfb25790c900c527969e78bd3c"}]`,
			EventName:        "OperatorSharesIncreased",
			LogIndex:         big.NewInt(400).Uint64(),
			OutputData:       `{"shares": "100", "strategy": "0x93c4b944d05dfe6df7645a86cd2206016c51564d"}`,
			CreatedAt:        time.Time{},
			UpdatedAt:        time.Time{},
			DeletedAt:        time.Time{},
		}

		stateChange, err := model.HandleStateChange(&log)
		assert.Nil(t, err)
		assert.NotNil(t, stateChange)

		err = model.CommitFinalState(blockNumber)
		assert.Nil(t, err)

		states := []OperatorShares{}
		statesRes := model.DB.
			Model(&OperatorShares{}).
			Raw("select * from operator_shares where block_number = @blockNumber", sql.Named("blockNumber", blockNumber)).
			Scan(&states)

		if statesRes.Error != nil {
			t.Fatalf("Failed to fetch operator_shares: %v", statesRes.Error)
		}
		assert.Equal(t, 1, len(states))

		assert.Equal(t, "100", states[0].Shares)

		stateRoot, err := model.GenerateStateRoot(blockNumber)
		assert.Nil(t, err)
		assert.True(t, len(stateRoot) > 0)

		t.Cleanup(func() {
			teardown(model)
		})
	})
	t.Run("Should handle state transition for operator shares decreased", func(t *testing.T) {
		esm := stateManager.NewEigenStateManager(l, grm)
		blockNumber := uint64(200)
		log := storage.TransactionLog{
			TransactionHash:  "some hash",
			TransactionIndex: big.NewInt(100).Uint64(),
			BlockNumber:      blockNumber,
			Address:          cfg.GetContractsMapForChain().DelegationManager,
			Arguments:        `[{"Name": "operator", "Type": "address", "Value": "0x32f766cf7BC7dEE7F65573587BECd7AdB2a5CC7f"}, {"Name": "staker", "Type": "address", "Value": ""}, {"Name": "strategy", "Type": "address", "Value": ""}, {"Name": "shares", "Type": "uint256", "Value": ""}]`,
			EventName:        "OperatorSharesDecreased",
			LogIndex:         big.NewInt(400).Uint64(),
			OutputData:       `{"shares": 1670000000000000000000, "staker": "0x32f766cf7bc7dee7f65573587becd7adb2a5cc7f", "strategy": "0x80528d6e9a2babfc766965e0e26d5ab08d9cfaf9"}`,
			CreatedAt:        time.Time{},
			UpdatedAt:        time.Time{},
			DeletedAt:        time.Time{},
		}

		model, err := NewOperatorSharesModel(esm, grm, l, cfg)
		assert.Nil(t, err)

		err = model.SetupStateForBlock(blockNumber)
		assert.Nil(t, err)

		stateChange, err := model.HandleStateChange(&log)
		assert.Nil(t, err)
		assert.NotNil(t, stateChange)

		stateChangeTyped := stateChange.(*AccumulatedStateChange)

		assert.Equal(t, "-1670000000000000000000", stateChangeTyped.Shares.String())
		assert.Equal(t, strings.ToLower("0x32f766cf7BC7dEE7F65573587BECd7AdB2a5CC7f"), stateChangeTyped.Operator)
		assert.Equal(t, "0x80528d6e9a2babfc766965e0e26d5ab08d9cfaf9", stateChangeTyped.Strategy)
	})
	t.Cleanup(func() {
		postgres.TeardownTestDatabase(dbName, cfg, grm, l)
	})
}
