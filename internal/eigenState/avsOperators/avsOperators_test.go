package avsOperators

import (
	"database/sql"
	"fmt"
	"github.com/Layr-Labs/sidecar/internal/config"
	"github.com/Layr-Labs/sidecar/internal/eigenState"
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
	*eigenState.EigenStateManager,
	error,
) {
	cfg := tests.GetConfig()
	l, _ := logger.NewLogger(&logger.LoggerConfig{Debug: cfg.Debug})

	_, grm, err := tests.GetDatabaseConnection(cfg)

	eigenState := eigenState.NewEigenStateManager(l)

	return cfg, grm, l, eigenState, err
}

func Test_AvsOperatorState(t *testing.T) {
	cfg, grm, l, esm, err := setup()

	if err != nil {
		t.Fatal(err)
	}

	t.Run("Should create a new AvsOperatorState", func(t *testing.T) {
		avsOperatorState, err := NewAvsOperators(esm, grm, cfg.Network, cfg.Environment, l, cfg)
		assert.Nil(t, err)
		assert.NotNil(t, avsOperatorState)
	})
	t.Run("Should register AvsOperatorState", func(t *testing.T) {
		log := storage.TransactionLog{
			TransactionHash:  "some hash",
			TransactionIndex: 100,
			BlockNumber:      200,
			BlockSequenceId:  300,
			Address:          "some address",
			Arguments:        "some arguments",
			EventName:        "OperatorAVSRegistrationStatusUpdated",
			LogIndex:         400,
			OutputData:       "some output data",
			CreatedAt:        time.Time{},
			UpdatedAt:        time.Time{},
			DeletedAt:        time.Time{},
		}

		avsOperatorState, err := NewAvsOperators(esm, grm, cfg.Network, cfg.Environment, l, cfg)
		fmt.Printf("avsOperatorState err: %+v\n", err)

		res, err := avsOperatorState.HandleStateChange(&log)
		assert.Nil(t, err)
		t.Logf("res_typed: %+v\n", res)

		avsOperatorState.Db.Raw("truncate table avs_operator_changes cascade").Scan(&res)
		avsOperatorState.Db.Raw("truncate table registered_avs_operators cascade").Scan(&res)
	})
	t.Run("Should register AvsOperatorState and generate the table for the block", func(t *testing.T) {
		log := storage.TransactionLog{
			TransactionHash:  "some hash",
			TransactionIndex: 100,
			BlockNumber:      200,
			BlockSequenceId:  300,
			Address:          "some address",
			Arguments:        "some arguments",
			EventName:        "OperatorAVSRegistrationStatusUpdated",
			LogIndex:         400,
			OutputData:       "some output data",
			CreatedAt:        time.Time{},
			UpdatedAt:        time.Time{},
			DeletedAt:        time.Time{},
		}

		avsOperatorState, err := NewAvsOperators(esm, grm, cfg.Network, cfg.Environment, l, cfg)
		assert.Nil(t, err)

		stateChange, err := avsOperatorState.HandleStateChange(&log)
		assert.Nil(t, err)
		fmt.Printf("stateChange: %+v\n", stateChange)

		err = avsOperatorState.WriteFinalState(200)
		assert.Nil(t, err)

		states := []RegisteredAvsOperators{}
		statesRes := avsOperatorState.Db.
			Model(&RegisteredAvsOperators{}).
			Raw("select * from registered_avs_operators where block_number = @blockNumber", sql.Named("blockNumber", 200)).
			Scan(&states)

		if statesRes.Error != nil {
			t.Fatalf("Failed to fetch registered_avs_operators: %v", statesRes.Error)
		}
		assert.Equal(t, 1, len(states))
		fmt.Printf("states: %+v\n", states)
	})
}
