package stakerDelegations

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

func teardown(model *StakerDelegationsModel) {
	model.Db.Exec("truncate table staker_delegation_changes cascade")
	model.Db.Exec("truncate table delegated_stakers cascade")
}

func Test_DelegatedStakersState(t *testing.T) {
	cfg, grm, l, err := setup()

	if err != nil {
		t.Fatal(err)
	}

	t.Run("Should create a new StakerDelegationsModel", func(t *testing.T) {
		esm := stateManager.NewEigenStateManager(l)
		model, err := NewStakerDelegationsModel(esm, grm, cfg.Network, cfg.Environment, l, cfg)
		assert.Nil(t, err)
		assert.NotNil(t, model)
	})
	t.Run("Should register StakerDelegationsModel", func(t *testing.T) {
		esm := stateManager.NewEigenStateManager(l)
		blockNumber := uint64(200)
		log := storage.TransactionLog{
			TransactionHash:  "some hash",
			TransactionIndex: 100,
			BlockNumber:      blockNumber,
			BlockSequenceId:  300,
			Address:          cfg.GetContractsMapForEnvAndNetwork().DelegationManager,
			Arguments:        `[{"Value": "0x5fc1b61816ddeb33b65a02a42b29059ecd3a20e9" }, { "Value": "0x5accc90436492f24e6af278569691e2c942a676d" }]`,
			EventName:        "StakerDelegated",
			LogIndex:         400,
			OutputData:       `{}`,
			CreatedAt:        time.Time{},
			UpdatedAt:        time.Time{},
			DeletedAt:        time.Time{},
		}

		model, err := NewStakerDelegationsModel(esm, grm, cfg.Network, cfg.Environment, l, cfg)

		assert.Equal(t, true, model.IsInterestingLog(&log))

		res, err := model.HandleStateChange(&log)
		assert.Nil(t, err)
		assert.NotNil(t, res)

		teardown(model)
	})
	t.Run("Should register StakerDelegationsModel and generate the table for the block", func(t *testing.T) {
		esm := stateManager.NewEigenStateManager(l)
		blockNumber := uint64(200)

		log := storage.TransactionLog{
			TransactionHash:  "some hash",
			TransactionIndex: 100,
			BlockNumber:      blockNumber,
			BlockSequenceId:  300,
			Address:          cfg.GetContractsMapForEnvAndNetwork().DelegationManager,
			Arguments:        `[{"Value": "0x5fc1b61816ddeb33b65a02a42b29059ecd3a20e9" }, { "Value": "0x5accc90436492f24e6af278569691e2c942a676d" }]`,
			EventName:        "StakerDelegated",
			LogIndex:         400,
			OutputData:       `{}`,
			CreatedAt:        time.Time{},
			UpdatedAt:        time.Time{},
			DeletedAt:        time.Time{},
		}

		model, err := NewStakerDelegationsModel(esm, grm, cfg.Network, cfg.Environment, l, cfg)
		assert.Nil(t, err)

		assert.Equal(t, true, model.IsInterestingLog(&log))

		stateChange, err := model.HandleStateChange(&log)
		assert.Nil(t, err)
		assert.NotNil(t, stateChange)

		err = model.WriteFinalState(blockNumber)
		assert.Nil(t, err)

		states := []DelegatedStakers{}
		statesRes := model.Db.
			Model(&DelegatedStakers{}).
			Raw("select * from delegated_stakers where block_number = @blockNumber", sql.Named("blockNumber", blockNumber)).
			Scan(&states)

		if statesRes.Error != nil {
			t.Fatalf("Failed to fetch delegated_stakers: %v", statesRes.Error)
		}
		assert.Equal(t, 1, len(states))

		stateRoot, err := model.GenerateStateRoot(blockNumber)
		assert.Nil(t, err)
		assert.True(t, len(stateRoot) > 0)

		teardown(model)
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
				BlockSequenceId:  300,
				Address:          cfg.GetContractsMapForEnvAndNetwork().DelegationManager,
				Arguments:        `[{"Value": "0x5fc1b61816ddeb33b65a02a42b29059ecd3a20e9" }, { "Value": "0x5accc90436492f24e6af278569691e2c942a676d" }]`,
				EventName:        "StakerDelegated",
				LogIndex:         400,
				OutputData:       `{}`,
				CreatedAt:        time.Time{},
				UpdatedAt:        time.Time{},
				DeletedAt:        time.Time{},
			},
			&storage.TransactionLog{
				TransactionHash:  "some hash",
				TransactionIndex: 100,
				BlockNumber:      blocks[1],
				BlockSequenceId:  300,
				Address:          cfg.GetContractsMapForEnvAndNetwork().DelegationManager,
				Arguments:        `[{"Value": "0x5fc1b61816ddeb33b65a02a42b29059ecd3a20e9" }, { "Value": "0x5accc90436492f24e6af278569691e2c942a676d" }]`,
				EventName:        "StakerUndelegated",
				LogIndex:         400,
				OutputData:       `{}`,
				CreatedAt:        time.Time{},
				UpdatedAt:        time.Time{},
				DeletedAt:        time.Time{},
			},
		}

		model, err := NewStakerDelegationsModel(esm, grm, cfg.Network, cfg.Environment, l, cfg)
		assert.Nil(t, err)

		for _, log := range logs {
			assert.True(t, model.IsInterestingLog(log))

			stateChange, err := model.HandleStateChange(log)
			assert.Nil(t, err)
			assert.NotNil(t, stateChange)

			err = model.WriteFinalState(log.BlockNumber)
			assert.Nil(t, err)

			states := []DelegatedStakers{}
			statesRes := model.Db.
				Model(&DelegatedStakers{}).
				Raw("select * from delegated_stakers where block_number = @blockNumber", sql.Named("blockNumber", log.BlockNumber)).
				Scan(&states)

			if statesRes.Error != nil {
				t.Fatalf("Failed to fetch delegated_stakers: %v", statesRes.Error)
			}

			if log.BlockNumber == blocks[0] {
				assert.Equal(t, 1, len(states))
				diffs, err := model.getDifferenceInStates(log.BlockNumber)
				assert.Nil(t, err)
				assert.Equal(t, 1, len(diffs))
				assert.Equal(t, true, diffs[0].Delegated)
			} else if log.BlockNumber == blocks[1] {
				assert.Equal(t, 0, len(states))
				diffs, err := model.getDifferenceInStates(log.BlockNumber)
				assert.Nil(t, err)
				assert.Equal(t, 1, len(diffs))
				assert.Equal(t, false, diffs[0].Delegated)
			}

			stateRoot, err := model.GenerateStateRoot(log.BlockNumber)
			assert.Nil(t, err)
			assert.True(t, len(stateRoot) > 0)
		}

		teardown(model)
	})
}
