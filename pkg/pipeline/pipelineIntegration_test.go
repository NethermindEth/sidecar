package pipeline

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/Layr-Labs/sidecar/pkg/clients/ethereum"
	"github.com/Layr-Labs/sidecar/pkg/contractCaller/sequentialContractCaller"
	"github.com/Layr-Labs/sidecar/pkg/contractManager"
	"github.com/Layr-Labs/sidecar/pkg/contractStore/postgresContractStore"
	"github.com/Layr-Labs/sidecar/pkg/eigenState"
	"github.com/Layr-Labs/sidecar/pkg/fetcher"
	"github.com/Layr-Labs/sidecar/pkg/indexer"
	"github.com/Layr-Labs/sidecar/pkg/postgres"
	"github.com/Layr-Labs/sidecar/pkg/rewards"
	"github.com/Layr-Labs/sidecar/pkg/storage"
	pgStorage "github.com/Layr-Labs/sidecar/pkg/storage/postgres"
	"log"
	"testing"

	"github.com/Layr-Labs/sidecar/internal/config"
	"github.com/Layr-Labs/sidecar/internal/logger"
	"github.com/Layr-Labs/sidecar/internal/metrics"
	"github.com/Layr-Labs/sidecar/internal/tests"
	"github.com/Layr-Labs/sidecar/pkg/eigenState/avsOperators"
	"github.com/Layr-Labs/sidecar/pkg/eigenState/stateManager"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func setup() (
	*fetcher.Fetcher,
	*indexer.Indexer,
	storage.BlockStore,
	*stateManager.EigenStateManager,
	*rewards.RewardsCalculator,
	*config.Config,
	*zap.Logger,
	*gorm.DB,
	string,
) {
	const (
		rpcUrl    = "https://tame-fabled-liquid.quiknode.pro/f27d4be93b4d7de3679f5c5ae881233f857407a0/"
		statsdUrl = "localhost:8125"
	)

	cfg := config.NewConfig()
	cfg.Chain = config.Chain_Mainnet
	cfg.EthereumRpcConfig.BaseUrl = rpcUrl
	cfg.StatsdUrl = statsdUrl
	cfg.DatabaseConfig = *tests.GetDbConfigFromEnv()

	l, _ := logger.NewLogger(&logger.LoggerConfig{Debug: true})

	sdc, err := metrics.InitStatsdClient(statsdUrl)
	if err != nil {
		l.Sugar().Fatal("Failed to setup statsd client", zap.Error(err))
	}

	client := ethereum.NewClient(rpcUrl, l)

	dbname, _, grm, err := postgres.GetTestPostgresDatabase(cfg.DatabaseConfig, l)
	if err != nil {
		log.Fatal(err)
	}

	contractStore := postgresContractStore.NewPostgresContractStore(grm, l, cfg)
	if err := contractStore.InitializeCoreContracts(); err != nil {
		log.Fatalf("Failed to initialize core contracts: %v", err)
	}

	cm := contractManager.NewContractManager(contractStore, client, sdc, l)

	mds := pgStorage.NewPostgresBlockStore(grm, l, cfg)

	sm := stateManager.NewEigenStateManager(l, grm)

	if err := eigenState.LoadEigenStateModels(sm, grm, l, cfg); err != nil {
		l.Sugar().Fatalw("Failed to load eigen state models", zap.Error(err))
	}

	rc, _ := rewards.NewRewardsCalculator(cfg, grm, mds, l)

	fetchr := fetcher.NewFetcher(client, cfg, l)

	cc := sequentialContractCaller.NewSequentialContractCaller(client, cfg, l)

	idxr := indexer.NewIndexer(mds, contractStore, cm, client, fetchr, cc, grm, l, cfg)

	return fetchr, idxr, mds, sm, rc, cfg, l, grm, dbname

}

func Test_PipelineIntegration(t *testing.T) {
	fetchr, idxr, mds, sm, rc, cfg, l, grm, dbName := setup()
	t.Run("Should create a new Pipeline", func(t *testing.T) {
		p := NewPipeline(fetchr, idxr, mds, sm, rc, cfg, l)
		assert.NotNil(t, p)
	})

	t.Run("Should index a block, transaction with logs", func(t *testing.T) {
		blockNumber := uint64(20386320)

		p := NewPipeline(fetchr, idxr, mds, sm, rc, cfg, l)

		err := p.RunForBlockBatch(context.Background(), blockNumber, blockNumber+1, true)
		assert.Nil(t, err)

		query := `select * from avs_operator_state_changes where block_number = @blockNumber`
		avsOperatorChanges := make([]avsOperators.AvsOperatorStateChange, 0)
		res := grm.Raw(query, sql.Named("blockNumber", blockNumber)).Scan(&avsOperatorChanges)
		assert.Nil(t, res.Error)

		for _, change := range avsOperatorChanges {
			fmt.Printf("Change: %+v\n", change)
		}

		assert.Equal(t, 1, len(avsOperatorChanges))
		assert.Equal(t, "0xf6ad76de4c80c056a51fcb457942df40a6d99f76", avsOperatorChanges[0].Operator)
		assert.Equal(t, "0xe7d0894ac9266f5cbe8f8e750ac6cbe128fbbeb7", avsOperatorChanges[0].Avs)
		assert.Equal(t, uint64(128), avsOperatorChanges[0].LogIndex)
		assert.Equal(t, blockNumber, avsOperatorChanges[0].BlockNumber)
	})
	t.Cleanup(func() {
		postgres.TeardownTestDatabase(dbName, cfg, grm, l)
	})
}
