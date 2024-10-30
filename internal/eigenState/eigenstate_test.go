package eigenState

import (
	"github.com/Layr-Labs/go-sidecar/internal/postgres"
	"testing"

	"github.com/Layr-Labs/go-sidecar/internal/config"
	"github.com/Layr-Labs/go-sidecar/internal/eigenState/avsOperators"
	"github.com/Layr-Labs/go-sidecar/internal/eigenState/operatorShares"
	"github.com/Layr-Labs/go-sidecar/internal/eigenState/stateManager"
	"github.com/Layr-Labs/go-sidecar/internal/logger"
	"github.com/Layr-Labs/go-sidecar/internal/tests"
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

func Test_EigenStateManager(t *testing.T) {
	dbName, grm, l, cfg, err := setup()

	if err != nil {
		t.Fatal(err)
	}

	t.Run("Should create a new EigenStateManager", func(t *testing.T) {
		esm := stateManager.NewEigenStateManager(l, grm)
		assert.NotNil(t, esm)
	})
	t.Run("Should create a state root with states from models", func(t *testing.T) {
		esm := stateManager.NewEigenStateManager(l, grm)
		avsOperatorsModel, err := avsOperators.NewAvsOperatorsModel(esm, grm, l, cfg)
		assert.Nil(t, err)
		assert.NotNil(t, avsOperatorsModel)

		operatorSharesModel, err := operatorShares.NewOperatorSharesModel(esm, grm, l, cfg)
		assert.Nil(t, err)
		assert.NotNil(t, operatorSharesModel)

		indexes := esm.GetSortedModelIndexes()
		assert.Equal(t, 2, len(indexes))
		assert.Equal(t, 0, indexes[0])
		assert.Equal(t, 1, indexes[1])

		err = esm.InitProcessingForBlock(200)
		assert.Nil(t, err)

		root, err := esm.GenerateStateRoot(200, "0x123")
		assert.Nil(t, err)
		assert.True(t, len(root) > 0)
	})
	t.Cleanup(func() {
		postgres.TeardownTestDatabase(dbName, cfg, grm, l)
	})
}
