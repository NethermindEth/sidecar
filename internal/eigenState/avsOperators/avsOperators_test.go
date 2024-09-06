package avsOperators

import (
	"database/sql"
	"github.com/Layr-Labs/sidecar/internal/config"
	"github.com/Layr-Labs/sidecar/internal/eigenState/stateManager"
	"github.com/Layr-Labs/sidecar/internal/logger"
	"github.com/Layr-Labs/sidecar/internal/storage"
	"github.com/Layr-Labs/sidecar/internal/tests"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"gorm.io/gorm"
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

	_, grm, err := tests.GetDatabaseConnection(cfg)

	return cfg, grm, l, err
}

func teardown(model *AvsOperatorsModel) {
	model.Db.Exec("truncate table avs_operator_changes cascade")
	model.Db.Exec("truncate table registered_avs_operators cascade")
}

func Test_AvsOperatorState(t *testing.T) {
	cfg, grm, l, err := setup()

	if err != nil {
		t.Fatal(err)
	}

	t.Run("Should create a new AvsOperatorState", func(t *testing.T) {
		esm := stateManager.NewEigenStateManager(l)
		avsOperatorState, err := NewAvsOperators(esm, grm, cfg.Network, cfg.Environment, l, cfg)
		assert.Nil(t, err)
		assert.NotNil(t, avsOperatorState)
	})
	t.Run("Should register AvsOperatorState", func(t *testing.T) {
		esm := stateManager.NewEigenStateManager(l)
		blockNumber := uint64(200)
		log := storage.TransactionLog{
			TransactionHash:  "some hash",
			TransactionIndex: 100,
			BlockNumber:      blockNumber,
			Address:          cfg.GetContractsMapForEnvAndNetwork().AvsDirectory,
			Arguments:        `[{"Value": "0xdf25bdcdcdd9a3dd8c9069306c4dba8d90dd8e8e" }, { "Value": "0x870679e138bcdf293b7ff14dd44b70fc97e12fc0" }]`,
			EventName:        "OperatorAVSRegistrationStatusUpdated",
			LogIndex:         400,
			OutputData:       `{ "status": 1 }`,
			CreatedAt:        time.Time{},
			UpdatedAt:        time.Time{},
			DeletedAt:        time.Time{},
		}

		avsOperatorState, err := NewAvsOperators(esm, grm, cfg.Network, cfg.Environment, l, cfg)

		assert.Equal(t, true, avsOperatorState.IsInterestingLog(&log))

		err = avsOperatorState.InitBlockProcessing(blockNumber)
		assert.Nil(t, err)

		res, err := avsOperatorState.HandleStateChange(&log)
		assert.Nil(t, err)
		assert.NotNil(t, res)

		teardown(avsOperatorState)
	})
	t.Run("Should register AvsOperatorState and generate the table for the block", func(t *testing.T) {
		esm := stateManager.NewEigenStateManager(l)
		blockNumber := uint64(200)

		log := storage.TransactionLog{
			TransactionHash:  "some hash",
			TransactionIndex: 100,
			BlockNumber:      blockNumber,
			Address:          cfg.GetContractsMapForEnvAndNetwork().AvsDirectory,
			Arguments:        `[{"Value": "0xdf25bdcdcdd9a3dd8c9069306c4dba8d90dd8e8e" }, { "Value": "0x870679e138bcdf293b7ff14dd44b70fc97e12fc0" }]`,
			EventName:        "OperatorAVSRegistrationStatusUpdated",
			LogIndex:         400,
			OutputData:       `{ "status": 1 }`,
			CreatedAt:        time.Time{},
			UpdatedAt:        time.Time{},
			DeletedAt:        time.Time{},
		}

		avsOperatorState, err := NewAvsOperators(esm, grm, cfg.Network, cfg.Environment, l, cfg)
		assert.Nil(t, err)

		assert.Equal(t, true, avsOperatorState.IsInterestingLog(&log))

		err = avsOperatorState.InitBlockProcessing(blockNumber)
		assert.Nil(t, err)

		stateChange, err := avsOperatorState.HandleStateChange(&log)
		assert.Nil(t, err)
		assert.NotNil(t, stateChange)

		err = avsOperatorState.CommitFinalState(blockNumber)
		assert.Nil(t, err)

		states := []RegisteredAvsOperators{}
		statesRes := avsOperatorState.Db.
			Model(&RegisteredAvsOperators{}).
			Raw("select * from registered_avs_operators where block_number = @blockNumber", sql.Named("blockNumber", blockNumber)).
			Scan(&states)

		if statesRes.Error != nil {
			t.Fatalf("Failed to fetch registered_avs_operators: %v", statesRes.Error)
		}
		assert.Equal(t, 1, len(states))

		stateRoot, err := avsOperatorState.GenerateStateRoot(blockNumber)
		assert.Nil(t, err)
		assert.True(t, len(stateRoot) > 0)

		teardown(avsOperatorState)
	})
	t.Run("Should correctly generate state across multiple blocks", func(t *testing.T) {
		esm := stateManager.NewEigenStateManager(l)
		blocks := []uint64{
			300,
			301,
		}

		logs := []*storage.TransactionLog{
			&storage.TransactionLog{
				TransactionHash:  "some hash",
				TransactionIndex: 100,
				BlockNumber:      blocks[0],
				Address:          cfg.GetContractsMapForEnvAndNetwork().AvsDirectory,
				Arguments:        `[{"Value": "0xdf25bdcdcdd9a3dd8c9069306c4dba8d90dd8e8e" }, { "Value": "0x870679e138bcdf293b7ff14dd44b70fc97e12fc0" }]`,
				EventName:        "OperatorAVSRegistrationStatusUpdated",
				LogIndex:         400,
				OutputData:       `{ "status": 1 }`,
				CreatedAt:        time.Time{},
				UpdatedAt:        time.Time{},
				DeletedAt:        time.Time{},
			},
			&storage.TransactionLog{
				TransactionHash:  "some hash",
				TransactionIndex: 100,
				BlockNumber:      blocks[1],
				Address:          cfg.GetContractsMapForEnvAndNetwork().AvsDirectory,
				Arguments:        `[{"Value": "0xdf25bdcdcdd9a3dd8c9069306c4dba8d90dd8e8e" }, { "Value": "0x870679e138bcdf293b7ff14dd44b70fc97e12fc0" }]`,
				EventName:        "OperatorAVSRegistrationStatusUpdated",
				LogIndex:         400,
				OutputData:       `{ "status": 0 }`,
				CreatedAt:        time.Time{},
				UpdatedAt:        time.Time{},
				DeletedAt:        time.Time{},
			},
		}

		avsOperatorState, err := NewAvsOperators(esm, grm, cfg.Network, cfg.Environment, l, cfg)
		assert.Nil(t, err)

		for _, log := range logs {
			assert.True(t, avsOperatorState.IsInterestingLog(log))

			err = avsOperatorState.InitBlockProcessing(log.BlockNumber)
			assert.Nil(t, err)

			stateChange, err := avsOperatorState.HandleStateChange(log)
			assert.Nil(t, err)
			assert.NotNil(t, stateChange)

			err = avsOperatorState.CommitFinalState(log.BlockNumber)
			assert.Nil(t, err)

			states := []RegisteredAvsOperators{}
			statesRes := avsOperatorState.Db.
				Model(&RegisteredAvsOperators{}).
				Raw("select * from registered_avs_operators where block_number = @blockNumber", sql.Named("blockNumber", log.BlockNumber)).
				Scan(&states)

			if statesRes.Error != nil {
				t.Fatalf("Failed to fetch registered_avs_operators: %v", statesRes.Error)
			}

			if log.BlockNumber == blocks[0] {
				assert.Equal(t, 1, len(states))
				inserts, deletes, err := avsOperatorState.prepareState(log.BlockNumber)
				assert.Nil(t, err)
				assert.Equal(t, 1, len(inserts))
				assert.Equal(t, 0, len(deletes))
			} else if log.BlockNumber == blocks[1] {
				assert.Equal(t, 0, len(states))
				inserts, deletes, err := avsOperatorState.prepareState(log.BlockNumber)
				assert.Nil(t, err)
				assert.Equal(t, 0, len(inserts))
				assert.Equal(t, 1, len(deletes))
			}

			stateRoot, err := avsOperatorState.GenerateStateRoot(log.BlockNumber)
			assert.Nil(t, err)
			assert.True(t, len(stateRoot) > 0)
		}

		teardown(avsOperatorState)
	})
}
