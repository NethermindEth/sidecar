package indexer

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"testing"
	"time"

	"os"

	"github.com/Layr-Labs/sidecar/internal/config"
	"github.com/Layr-Labs/sidecar/internal/logger"
	"github.com/Layr-Labs/sidecar/internal/metrics"
	"github.com/Layr-Labs/sidecar/internal/tests"
	"github.com/Layr-Labs/sidecar/pkg/abiFetcher"
	"github.com/Layr-Labs/sidecar/pkg/abiSource"
	"github.com/Layr-Labs/sidecar/pkg/clients/ethereum"
	"github.com/Layr-Labs/sidecar/pkg/contractCaller/multicallContractCaller"
	"github.com/Layr-Labs/sidecar/pkg/contractCaller/sequentialContractCaller"
	"github.com/Layr-Labs/sidecar/pkg/contractManager"
	"github.com/Layr-Labs/sidecar/pkg/contractStore/postgresContractStore"
	"github.com/Layr-Labs/sidecar/pkg/fetcher"
	"github.com/Layr-Labs/sidecar/pkg/postgres"
	"github.com/Layr-Labs/sidecar/pkg/storage"
	pgStorage "github.com/Layr-Labs/sidecar/pkg/storage/postgres"
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
	cfg.Chain = config.Chain_Holesky
	cfg.Debug = os.Getenv(config.Debug) == "true"
	cfg.DatabaseConfig = *tests.GetDbConfigFromEnv()

	l, _ := logger.NewLogger(&logger.LoggerConfig{Debug: cfg.Debug})

	dbname, _, grm, err := postgres.GetTestPostgresDatabase(cfg.DatabaseConfig, cfg, l)
	if err != nil {
		return dbname, nil, nil, nil, err
	}

	return dbname, grm, l, cfg, nil
}

func teardown(grm *gorm.DB) {
	queries := []string{
		`truncate table blocks cascade`,
		`truncate table operator_restaked_strategies`,
	}
	for _, query := range queries {
		res := grm.Exec(query)
		if res.Error != nil {
			fmt.Printf("Failed to run query: %v\n", res.Error)
		}
	}
}

func Test_IndexerRestakedStrategies(t *testing.T) {
	dbName, grm, l, cfg, err := setup()

	if err != nil {
		t.Fatal(err)
	}

	baseUrl := "https://winter-white-crater.ethereum-holesky.quiknode.pro/1b1d75c4ada73b7ad98e1488880649d4ea637733/"
	ethConfig := ethereum.DefaultNativeCallEthereumClientConfig()
	ethConfig.BaseUrl = baseUrl

	client := ethereum.NewClient(ethConfig, l)

	af := abiFetcher.NewAbiFetcher(client, &http.Client{Timeout: 5 * time.Second}, l, cfg, []abiSource.AbiSource{})

	metricsClients, err := metrics.InitMetricsSinksFromConfig(cfg, l)
	if err != nil {
		l.Sugar().Fatal("Failed to setup metrics sink", zap.Error(err))
	}

	sdc, err := metrics.NewMetricsSink(&metrics.MetricsSinkConfig{}, metricsClients)
	if err != nil {
		l.Sugar().Fatal("Failed to setup metrics sink", zap.Error(err))
	}

	contractStore := postgresContractStore.NewPostgresContractStore(grm, l, cfg)
	if err := contractStore.InitializeCoreContracts(); err != nil {
		log.Fatalf("Failed to initialize core contracts: %v", err)
	}

	mds := pgStorage.NewPostgresBlockStore(grm, l, cfg)

	fetchr := fetcher.NewFetcher(client, cfg, l)

	mccc := multicallContractCaller.NewMulticallContractCaller(client, l)

	scc := sequentialContractCaller.NewSequentialContractCaller(client, cfg, 10, l)

	cm := contractManager.NewContractManager(contractStore, client, af, sdc, l)

	t.Run("Integration - gets restaked strategies for avs/operator with multicall contract caller", func(t *testing.T) {
		avs := "0xD4A7E1Bd8015057293f0D0A557088c286942e84b"
		operator := "0xA8C128BD6f5A314b46202Dd7C68E7E2422eD61F2"

		block := &storage.Block{
			Number:    uint64(1191600),
			Hash:      "",
			BlockTime: time.Unix(1726063248, 0),
		}
		res := grm.Model(&storage.Block{}).Create(&block)
		assert.Nil(t, res.Error)

		contracts := cfg.GetContractsMapForChain()

		avsOperator := []*storage.ActiveAvsOperator{
			&storage.ActiveAvsOperator{
				Avs:      avs,
				Operator: operator,
			},
		}

		idxr := NewIndexer(mds, contractStore, cm, client, fetchr, mccc, grm, l, cfg)

		err = idxr.getAndInsertRestakedStrategies(context.Background(), avsOperator, contracts.AvsDirectory, block)
		assert.Nil(t, err)

		results := make([]storage.OperatorRestakedStrategies, 0)
		query := `select * from operator_restaked_strategies`
		result := grm.Raw(query).Scan(&results)

		assert.Nil(t, result.Error)
		assert.True(t, len(results) > 0)

		teardown(grm)
	})
	t.Run("Integration - gets restaked strategies for avs/operator with sequential contract caller", func(t *testing.T) {
		avs := "0xD4A7E1Bd8015057293f0D0A557088c286942e84b"
		operator := "0xA8C128BD6f5A314b46202Dd7C68E7E2422eD61F2"

		block := &storage.Block{
			Number:    uint64(1191600),
			Hash:      "",
			BlockTime: time.Unix(1726063248, 0),
		}
		res := grm.Model(&storage.Block{}).Create(&block)
		assert.Nil(t, res.Error)

		contracts := cfg.GetContractsMapForChain()

		avsOperator := []*storage.ActiveAvsOperator{
			&storage.ActiveAvsOperator{
				Avs:      avs,
				Operator: operator,
			},
		}

		idxr := NewIndexer(mds, contractStore, cm, client, fetchr, scc, grm, l, cfg)

		err = idxr.getAndInsertRestakedStrategies(context.Background(), avsOperator, contracts.AvsDirectory, block)
		assert.Nil(t, err)

		results := make([]storage.OperatorRestakedStrategies, 0)
		query := `select * from operator_restaked_strategies`
		result := grm.Raw(query).Scan(&results)

		assert.Nil(t, result.Error)
		assert.True(t, len(results) > 0)

		teardown(grm)
	})

	t.Cleanup(func() {
		postgres.TeardownTestDatabase(dbName, cfg, grm, l)
	})
}
