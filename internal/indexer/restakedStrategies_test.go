package indexer

import (
	"context"
	"fmt"
	"github.com/Layr-Labs/go-sidecar/internal/clients/ethereum"
	"github.com/Layr-Labs/go-sidecar/internal/clients/etherscan"
	"github.com/Layr-Labs/go-sidecar/internal/config"
	"github.com/Layr-Labs/go-sidecar/internal/contractCaller"
	"github.com/Layr-Labs/go-sidecar/internal/contractManager"
	"github.com/Layr-Labs/go-sidecar/internal/contractStore/sqliteContractStore"
	"github.com/Layr-Labs/go-sidecar/internal/fetcher"
	"github.com/Layr-Labs/go-sidecar/internal/logger"
	"github.com/Layr-Labs/go-sidecar/internal/metrics"
	"github.com/Layr-Labs/go-sidecar/internal/sqlite/migrations"
	"github.com/Layr-Labs/go-sidecar/internal/storage"
	sqliteBlockStore "github.com/Layr-Labs/go-sidecar/internal/storage/sqlite"
	"github.com/Layr-Labs/go-sidecar/internal/tests"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"log"
	"testing"
	"time"
)

var previousEnv = make(map[string]string)

func setup() (
	*config.Config,
	*gorm.DB,
	*zap.Logger,
	error,
) {
	tests.ReplaceEnv(map[string]string{
		"SIDECAR_ENVIRONMENT":           "testnet",
		"SIDECAR_NETWORK":               "holesky",
		"SIDECAR_ETHEREUM_RPC_BASE_URL": "http://54.198.82.217:8545",
		"SIDECAR_ETHERSCAN_API_KEYS":    "QIPXW3YCXPR5NQ9GXTRQ3TSXB9EKMGDE34",
		"SIDECAR_STATSD_URL":            "localhost:8125",
		"SIDECAR_DEBUG":                 "true",
	}, &previousEnv)
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

func teardown(grm *gorm.DB) {
	queries := []string{
		`delete from operator_restaked_strategies`,
	}
	for _, query := range queries {
		res := grm.Exec(query)
		if res.Error != nil {
			fmt.Printf("Failed to run query: %v\n", res.Error)
		}
	}
}

func Test_IndexerRestakedStrategies(t *testing.T) {
	cfg, grm, l, err := setup()

	if err != nil {
		t.Fatal(err)
	}

	client := ethereum.NewClient(cfg.EthereumRpcConfig.BaseUrl, l)
	sdc, err := metrics.InitStatsdClient(cfg.StatsdUrl)

	contractStore := sqliteContractStore.NewSqliteContractStore(grm, l, cfg)
	if err := contractStore.InitializeCoreContracts(); err != nil {
		log.Fatalf("Failed to initialize core contracts: %v", err)
	}

	mds := sqliteBlockStore.NewSqliteBlockStore(grm, l, cfg)

	fetchr := fetcher.NewFetcher(client, cfg, l)

	cc := contractCaller.NewContractCaller(client, l)

	etherscanClient := etherscan.NewEtherscanClient(cfg, l)

	cm := contractManager.NewContractManager(contractStore, etherscanClient, client, sdc, l)

	idxr := NewIndexer(mds, contractStore, etherscanClient, cm, client, fetchr, cc, l, cfg)

	t.Run("Integration - gets restaked strategies for avs/operator", func(t *testing.T) {
		avs := "0x5e78eff26480a75e06ccdabe88eb522d4d8e1c9d"
		operator := "0x000dba91cdee891ac56f9798f57f563257ea9ec8"

		block := &storage.Block{
			Number:    uint64(2314800),
			Hash:      "",
			BlockTime: time.Unix(1726063248, 0),
		}

		contracts := cfg.GetContractsMapForEnvAndNetwork()

		avsOperator := &storage.ActiveAvsOperator{
			Avs:      avs,
			Operator: operator,
		}

		err = idxr.getRestakedStrategiesForAvsOperator(context.Background(), contracts.AvsDirectory, avsOperator, block)
		assert.Nil(t, err)

		results := make([]storage.OperatorRestakedStrategies, 0)
		query := `select * from operator_restaked_strategies`
		result := grm.Raw(query).Scan(&results)

		assert.Nil(t, result.Error)
		assert.True(t, len(results) > 0)

		fmt.Printf("Results: %+v\n", results)

		t.Cleanup(func() {
			teardown(grm)
		})
	})
	t.Run("Integration - process avs/operators with multicall", func(t *testing.T) {
		avs := "0x5e78eff26480a75e06ccdabe88eb522d4d8e1c9d"
		operator := "0x000dba91cdee891ac56f9798f57f563257ea9ec8"

		block := &storage.Block{
			Number:    uint64(2314800),
			Hash:      "",
			BlockTime: time.Unix(1726063248, 0),
		}

		contracts := cfg.GetContractsMapForEnvAndNetwork()

		avsOperator := &storage.ActiveAvsOperator{
			Avs:      avs,
			Operator: operator,
		}

		err = idxr.getAndInsertRestakedStrategiesWithMulticall(context.Background(), []*storage.ActiveAvsOperator{avsOperator}, contracts.AvsDirectory, block)
		assert.Nil(t, err)

		results := make([]storage.OperatorRestakedStrategies, 0)
		query := `select * from operator_restaked_strategies`
		result := grm.Raw(query).Scan(&results)

		assert.Nil(t, result.Error)
		assert.True(t, len(results) > 0)

		fmt.Printf("Results: %+v\n", results)

		t.Cleanup(func() {
			teardown(grm)
		})
	})
	t.Run("Integration - gets restaked strategies for avs/operator multicall", func(t *testing.T) {
		avs := "0x5e78eff26480a75e06ccdabe88eb522d4d8e1c9d"
		operator := "0x000dba91cdee891ac56f9798f57f563257ea9ec8"

		block := &storage.Block{
			Number:    uint64(2314800),
			Hash:      "",
			BlockTime: time.Unix(1726063248, 0),
		}

		avsOperator := []*contractCaller.OperatorRestakedStrategy{
			&contractCaller.OperatorRestakedStrategy{
				Avs:      avs,
				Operator: operator,
			},
		}

		results, err := cc.GetOperatorRestakedStrategiesMulticall(context.Background(), avsOperator, block.Number)
		assert.Nil(t, err)

		assert.Equal(t, len(avsOperator), len(results))

		t.Cleanup(func() {
			teardown(grm)
		})
	})

	t.Cleanup(func() {
		teardown(grm)
		tests.RestoreEnv(previousEnv)
	})
}
