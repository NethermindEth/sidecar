package pipeline

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/Layr-Labs/go-sidecar/internal/contractStore/postgresContractStore"
	"github.com/Layr-Labs/go-sidecar/internal/postgres"
	postgres2 "github.com/Layr-Labs/go-sidecar/internal/storage/postgres"
	"log"
	"testing"

	"github.com/Layr-Labs/go-sidecar/internal/clients/ethereum"
	"github.com/Layr-Labs/go-sidecar/internal/clients/etherscan"
	"github.com/Layr-Labs/go-sidecar/internal/config"
	"github.com/Layr-Labs/go-sidecar/internal/contractCaller"
	"github.com/Layr-Labs/go-sidecar/internal/contractManager"
	"github.com/Layr-Labs/go-sidecar/internal/eigenState/avsOperators"
	"github.com/Layr-Labs/go-sidecar/internal/eigenState/operatorShares"
	"github.com/Layr-Labs/go-sidecar/internal/eigenState/rewardSubmissions"
	"github.com/Layr-Labs/go-sidecar/internal/eigenState/stakerDelegations"
	"github.com/Layr-Labs/go-sidecar/internal/eigenState/stakerShares"
	"github.com/Layr-Labs/go-sidecar/internal/eigenState/stateManager"
	"github.com/Layr-Labs/go-sidecar/internal/eigenState/submittedDistributionRoots"
	"github.com/Layr-Labs/go-sidecar/internal/fetcher"
	"github.com/Layr-Labs/go-sidecar/internal/indexer"
	"github.com/Layr-Labs/go-sidecar/internal/logger"
	"github.com/Layr-Labs/go-sidecar/internal/metrics"
	"github.com/Layr-Labs/go-sidecar/internal/storage"
	"github.com/Layr-Labs/go-sidecar/internal/tests"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func setup() (
	*fetcher.Fetcher,
	*indexer.Indexer,
	storage.BlockStore,
	*stateManager.EigenStateManager,
	*config.Config,
	*zap.Logger,
	*gorm.DB,
	string,
) {
	const (
		rpcUrl    = "http://54.198.82.217:8545"
		statsdUrl = "localhost:8125"
	)
	etherscanApiKeys := []string{"SOME KEY"}

	cfg := config.NewConfig()
	cfg.EthereumRpcConfig.BaseUrl = rpcUrl
	cfg.StatsdUrl = statsdUrl
	cfg.EtherscanConfig.ApiKeys = etherscanApiKeys
	cfg.DatabaseConfig = *tests.GetDbConfigFromEnv()

	l, _ := logger.NewLogger(&logger.LoggerConfig{Debug: true})

	sdc, err := metrics.InitStatsdClient(statsdUrl)
	if err != nil {
		l.Sugar().Fatal("Failed to setup statsd client", zap.Error(err))
	}

	etherscanClient := etherscan.NewEtherscanClient(&config.Config{
		EtherscanConfig: config.EtherscanConfig{
			ApiKeys: etherscanApiKeys,
		},
		Chain: "holesky",
	}, l)
	client := ethereum.NewClient(rpcUrl, l)

	dbname, _, grm, err := postgres.GetTestPostgresDatabase(cfg.DatabaseConfig, l)
	if err != nil {
		log.Fatal(err)
	}

	contractStore := postgresContractStore.NewPostgresContractStore(grm, l, cfg)
	if err := contractStore.InitializeCoreContracts(); err != nil {
		log.Fatalf("Failed to initialize core contracts: %v", err)
	}

	cm := contractManager.NewContractManager(contractStore, etherscanClient, client, sdc, l)

	mds := postgres2.NewPostgresBlockStore(grm, l, cfg)

	sm := stateManager.NewEigenStateManager(l, grm)

	if _, err := avsOperators.NewAvsOperatorsModel(sm, grm, l, cfg); err != nil {
		l.Sugar().Fatalw("Failed to create AvsOperatorsModel", zap.Error(err))
	}
	if _, err := operatorShares.NewOperatorSharesModel(sm, grm, l, cfg); err != nil {
		l.Sugar().Fatalw("Failed to create OperatorSharesModel", zap.Error(err))
	}
	if _, err := stakerDelegations.NewStakerDelegationsModel(sm, grm, l, cfg); err != nil {
		l.Sugar().Fatalw("Failed to create StakerDelegationsModel", zap.Error(err))
	}
	if _, err := stakerShares.NewStakerSharesModel(sm, grm, l, cfg); err != nil {
		l.Sugar().Fatalw("Failed to create StakerSharesModel", zap.Error(err))
	}
	if _, err := submittedDistributionRoots.NewSubmittedDistributionRootsModel(sm, grm, l, cfg); err != nil {
		l.Sugar().Fatalw("Failed to create SubmittedDistributionRootsModel", zap.Error(err))
	}
	if _, err := rewardSubmissions.NewRewardSubmissionsModel(sm, grm, l, cfg); err != nil {
		l.Sugar().Fatalw("Failed to create RewardSubmissionsModel", zap.Error(err))
	}

	fetchr := fetcher.NewFetcher(client, cfg, l)

	cc := contractCaller.NewContractCaller(client, l)

	idxr := indexer.NewIndexer(mds, contractStore, etherscanClient, cm, client, fetchr, cc, l, cfg)

	return fetchr, idxr, mds, sm, cfg, l, grm, dbname

}

func Test_Pipeline_Integration(t *testing.T) {
	fetchr, idxr, mds, sm, cfg, l, grm, dbName := setup()
	t.Run("Should create a new Pipeline", func(t *testing.T) {
		p := NewPipeline(fetchr, idxr, mds, sm, l)
		assert.NotNil(t, p)
	})

	t.Run("Should index a block, transaction with logs", func(t *testing.T) {
		blockNumber := uint64(1175560)
		transactionHash := "0x78cc56f0700e7ba5055f124243e6591fc1199cf3c75a17d50f8ea438254c9a76"
		logIndex := uint64(14)

		fmt.Printf("transactionHash: %s %d\n", transactionHash, logIndex)

		p := NewPipeline(fetchr, idxr, mds, sm, l)

		err := p.RunForBlock(context.Background(), blockNumber)
		assert.Nil(t, err)

		query := `select * from delegated_stakers where block_number = @blockNumber`
		delegatedStakers := make([]stakerDelegations.DelegatedStakers, 0)
		res := grm.Raw(query, sql.Named("blockNumber", blockNumber)).Scan(&delegatedStakers)
		assert.Nil(t, res.Error)

		assert.Equal(t, 1, len(delegatedStakers))
	})
	t.Cleanup(func() {
		postgres.TeardownTestDatabase(dbName, cfg, grm, l)
	})
}
