package stakerDelegations

import (
	"database/sql"
	"github.com/Layr-Labs/go-sidecar/pkg/postgres"
	"github.com/Layr-Labs/go-sidecar/pkg/storage"
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

func teardown(model *StakerDelegationsModel) {
	model.DB.Exec("truncate table staker_delegation_changes cascade")
	model.DB.Exec("truncate table delegated_stakers cascade")
	model.DB.Exec("truncate table staker_delegation_changes cascade")
}

func Test_DelegatedStakersState(t *testing.T) {
	dbName, grm, l, cfg, err := setup()

	if err != nil {
		t.Fatal(err)
	}

	t.Run("Should create a new StakerDelegationsModel", func(t *testing.T) {
		esm := stateManager.NewEigenStateManager(l, grm)
		model, err := NewStakerDelegationsModel(esm, grm, l, cfg)
		assert.Nil(t, err)
		assert.NotNil(t, model)
	})
	t.Run("Should register StakerDelegationsModel", func(t *testing.T) {
		esm := stateManager.NewEigenStateManager(l, grm)
		blockNumber := uint64(200)
		block := &storage.Block{
			Number:    blockNumber,
			Hash:      "",
			BlockTime: time.Unix(1726063248, 0),
		}
		res := grm.Model(&storage.Block{}).Create(&block)
		assert.Nil(t, res.Error)

		log := storage.TransactionLog{
			TransactionHash:  "some hash",
			TransactionIndex: 100,
			BlockNumber:      blockNumber,
			Address:          cfg.GetContractsMapForChain().DelegationManager,
			Arguments:        `[{"Name":"staker","Type":"address","Value":"0xbde83df53bc7d159700e966ad5d21e8b7c619459","Indexed":true},{"Name":"operator","Type":"address","Value":"0xbde83df53bc7d159700e966ad5d21e8b7c619459","Indexed":true}]`,
			EventName:        "StakerDelegated",
			LogIndex:         400,
			OutputData:       `{}`,
			CreatedAt:        time.Time{},
			UpdatedAt:        time.Time{},
			DeletedAt:        time.Time{},
		}

		model, err := NewStakerDelegationsModel(esm, grm, l, cfg)
		assert.Nil(t, err)

		assert.Equal(t, true, model.IsInterestingLog(&log))

		err = model.SetupStateForBlock(blockNumber)
		assert.Nil(t, err)

		result, err := model.HandleStateChange(&log)
		assert.Nil(t, err)
		assert.NotNil(t, res)

		typedChange := result.(*AccumulatedStateChange)
		assert.Equal(t, "0xbde83df53bc7d159700e966ad5d21e8b7c619459", typedChange.Staker)
		assert.Equal(t, "0xbde83df53bc7d159700e966ad5d21e8b7c619459", typedChange.Operator)

		t.Cleanup(func() {
			teardown(model)
		})
	})
	t.Run("Should register StakerDelegationsModel and generate the table for the block", func(t *testing.T) {
		esm := stateManager.NewEigenStateManager(l, grm)
		blockNumber := uint64(201)
		block := &storage.Block{
			Number:    blockNumber,
			Hash:      "",
			BlockTime: time.Unix(1726063248, 0),
		}
		res := grm.Model(&storage.Block{}).Create(&block)
		assert.Nil(t, res.Error)

		log := storage.TransactionLog{
			TransactionHash:  "some hash",
			TransactionIndex: 100,
			BlockNumber:      blockNumber,
			Address:          cfg.GetContractsMapForChain().DelegationManager,
			Arguments:        `[{"Name":"staker","Type":"address","Value":"0xbde83df53bc7d159700e966ad5d21e8b7c619459","Indexed":true},{"Name":"operator","Type":"address","Value":"0xbde83df53bc7d159700e966ad5d21e8b7c619459","Indexed":true}]`,
			EventName:        "StakerDelegated",
			LogIndex:         400,
			OutputData:       `{}`,
			CreatedAt:        time.Time{},
			UpdatedAt:        time.Time{},
			DeletedAt:        time.Time{},
		}

		model, err := NewStakerDelegationsModel(esm, grm, l, cfg)
		assert.Nil(t, err)

		assert.Equal(t, true, model.IsInterestingLog(&log))

		err = model.SetupStateForBlock(blockNumber)
		assert.Nil(t, err)

		stateChange, err := model.HandleStateChange(&log)
		assert.Nil(t, err)
		assert.NotNil(t, stateChange)

		typedChange := stateChange.(*AccumulatedStateChange)
		assert.Equal(t, "0xbde83df53bc7d159700e966ad5d21e8b7c619459", typedChange.Staker)
		assert.Equal(t, "0xbde83df53bc7d159700e966ad5d21e8b7c619459", typedChange.Operator)

		err = model.CommitFinalState(blockNumber)
		assert.Nil(t, err)

		states := []StakerDelegationChange{}
		statesRes := model.DB.
			Model(&StakerDelegationChange{}).
			Raw("select * from staker_delegation_changes where block_number = @blockNumber", sql.Named("blockNumber", blockNumber)).
			Scan(&states)

		if statesRes.Error != nil {
			t.Fatalf("Failed to fetch delegated_stakers: %v", statesRes.Error)
		}
		assert.Equal(t, 1, len(states))

		stateRoot, err := model.GenerateStateRoot(blockNumber)
		assert.Nil(t, err)
		assert.True(t, len(stateRoot) > 0)

		t.Cleanup(func() {
			teardown(model)
		})
	})
	t.Run("Should correctly generate state across multiple blocks", func(t *testing.T) {
		esm := stateManager.NewEigenStateManager(l, grm)
		blocks := []uint64{
			300,
			301,
		}

		for _, blockNumber := range blocks {
			block := &storage.Block{
				Number:    blockNumber,
				Hash:      "",
				BlockTime: time.Unix(1726063248, 0),
			}
			res := grm.Model(&storage.Block{}).Create(&block)
			assert.Nil(t, res.Error)
		}

		logs := []*storage.TransactionLog{
			{
				TransactionHash:  "some hash",
				TransactionIndex: 100,
				BlockNumber:      blocks[0],
				Address:          cfg.GetContractsMapForChain().DelegationManager,
				Arguments:        `[{"Name":"staker","Type":"address","Value":"0xbde83df53bc7d159700e966ad5d21e8b7c619459","Indexed":true},{"Name":"operator","Type":"address","Value":"0xbde83df53bc7d159700e966ad5d21e8b7c619459","Indexed":true}]`,
				EventName:        "StakerDelegated",
				LogIndex:         400,
				OutputData:       `{}`,
				CreatedAt:        time.Time{},
				UpdatedAt:        time.Time{},
				DeletedAt:        time.Time{},
			},
			{
				TransactionHash:  "some hash",
				TransactionIndex: 100,
				BlockNumber:      blocks[1],
				Address:          cfg.GetContractsMapForChain().DelegationManager,
				Arguments:        `[{"Name":"staker","Type":"address","Value":"0xbde83df53bc7d159700e966ad5d21e8b7c619459","Indexed":true},{"Name":"operator","Type":"address","Value":"0xbde83df53bc7d159700e966ad5d21e8b7c619459","Indexed":true}]`,
				EventName:        "StakerUndelegated",
				LogIndex:         400,
				OutputData:       `{}`,
				CreatedAt:        time.Time{},
				UpdatedAt:        time.Time{},
				DeletedAt:        time.Time{},
			},
		}

		model, err := NewStakerDelegationsModel(esm, grm, l, cfg)
		assert.Nil(t, err)

		for _, log := range logs {
			assert.True(t, model.IsInterestingLog(log))

			err = model.SetupStateForBlock(log.BlockNumber)
			assert.Nil(t, err)

			stateChange, err := model.HandleStateChange(log)
			assert.Nil(t, err)
			assert.NotNil(t, stateChange)

			err = model.CommitFinalState(log.BlockNumber)
			assert.Nil(t, err)

			states := []StakerDelegationChange{}
			statesRes := model.DB.
				Model(&StakerDelegationChange{}).
				Raw("select * from staker_delegation_changes where block_number = @blockNumber", sql.Named("blockNumber", log.BlockNumber)).
				Scan(&states)

			if statesRes.Error != nil {
				t.Fatalf("Failed to fetch delegated_stakers: %v", statesRes.Error)
			}

			if log.BlockNumber == blocks[0] {
				assert.Equal(t, 1, len(states))
				inserts, deletes, err := model.prepareState(log.BlockNumber)
				assert.Nil(t, err)
				assert.Equal(t, 1, len(inserts))
				assert.Equal(t, 0, len(deletes))
			} else if log.BlockNumber == blocks[1] {
				assert.Equal(t, 1, len(states))
				inserts, deletes, err := model.prepareState(log.BlockNumber)
				assert.Nil(t, err)
				assert.Equal(t, 0, len(inserts))
				assert.Equal(t, 1, len(deletes))
			}

			stateRoot, err := model.GenerateStateRoot(log.BlockNumber)
			assert.Nil(t, err)
			assert.True(t, len(stateRoot) > 0)
		}

		t.Cleanup(func() {
			teardown(model)
		})
	})
	t.Cleanup(func() {
		postgres.TeardownTestDatabase(dbName, cfg, grm, l)
	})
}
