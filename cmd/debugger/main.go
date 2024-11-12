package main

import (
	"context"
	"fmt"
	"github.com/Layr-Labs/go-sidecar/pkg/clients/ethereum"
	"github.com/Layr-Labs/go-sidecar/pkg/contractCaller/sequentialContractCaller"
	"github.com/Layr-Labs/go-sidecar/pkg/contractManager"
	"github.com/Layr-Labs/go-sidecar/pkg/contractStore/postgresContractStore"
	"github.com/Layr-Labs/go-sidecar/pkg/eigenState/rewardSubmissions"
	"github.com/Layr-Labs/go-sidecar/pkg/eigenState/submittedDistributionRoots"
	"github.com/Layr-Labs/go-sidecar/pkg/fetcher"
	"github.com/Layr-Labs/go-sidecar/pkg/indexer"
	"github.com/Layr-Labs/go-sidecar/pkg/pipeline"
	"github.com/Layr-Labs/go-sidecar/pkg/postgres"
	"github.com/Layr-Labs/go-sidecar/pkg/rewards"
	"github.com/Layr-Labs/go-sidecar/pkg/sidecar"
	pgStorage "github.com/Layr-Labs/go-sidecar/pkg/storage/postgres"
	"log"

	"github.com/Layr-Labs/go-sidecar/internal/config"
	"github.com/Layr-Labs/go-sidecar/internal/logger"
	"github.com/Layr-Labs/go-sidecar/internal/metrics"
	"github.com/Layr-Labs/go-sidecar/pkg/eigenState/avsOperators"
	"github.com/Layr-Labs/go-sidecar/pkg/eigenState/operatorShares"
	"github.com/Layr-Labs/go-sidecar/pkg/eigenState/stakerDelegations"
	"github.com/Layr-Labs/go-sidecar/pkg/eigenState/stakerShares"
	"github.com/Layr-Labs/go-sidecar/pkg/eigenState/stateManager"
	"github.com/Layr-Labs/go-sidecar/pkg/postgres/migrations"
	"go.uber.org/zap"
)

func main() {
	ctx := context.Background()
	cfg := config.NewConfig()

	l, _ := logger.NewLogger(&logger.LoggerConfig{Debug: cfg.Debug})

	sdc, err := metrics.InitStatsdClient(cfg.StatsdUrl)
	if err != nil {
		l.Sugar().Fatal("Failed to setup statsd client", zap.Error(err))
	}

	client := ethereum.NewClient(cfg.EthereumRpcConfig.BaseUrl, l)

	pgConfig := postgres.PostgresConfigFromDbConfig(&cfg.DatabaseConfig)

	pg, err := postgres.NewPostgres(pgConfig)
	if err != nil {
		l.Fatal("Failed to setup postgres connection", zap.Error(err))
	}

	grm, err := postgres.NewGormFromPostgresConnection(pg.Db)
	if err != nil {
		l.Fatal("Failed to create gorm instance", zap.Error(err))
	}

	migrator := migrations.NewMigrator(pg.Db, grm, l)
	if err = migrator.MigrateAll(); err != nil {
		l.Fatal("Failed to migrate", zap.Error(err))
	}

	contractStore := postgresContractStore.NewPostgresContractStore(grm, l, cfg)
	if err := contractStore.InitializeCoreContracts(); err != nil {
		log.Fatalf("Failed to initialize core contracts: %v", err)
	}

	cm := contractManager.NewContractManager(contractStore, client, sdc, l)

	mds := pgStorage.NewPostgresBlockStore(grm, l, cfg)
	if err != nil {
		log.Fatalln(err)
	}

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

	cc := sequentialContractCaller.NewSequentialContractCaller(client, cfg, l)

	idxr := indexer.NewIndexer(mds, contractStore, cm, client, fetchr, cc, grm, l, cfg)

	rc, err := rewards.NewRewardsCalculator(cfg, grm, mds, l)
	if err != nil {
		l.Sugar().Fatalw("Failed to create rewards calculator", zap.Error(err))
	}

	p := pipeline.NewPipeline(fetchr, idxr, mds, sm, rc, cfg, l)

	// Create new sidecar instance
	sidecar := sidecar.NewSidecar(&sidecar.SidecarConfig{
		GenesisBlockNumber: cfg.GetGenesisBlockNumber(),
	}, cfg, mds, p, sm, rc, l, client)

	// RPC channel to notify the RPC server to shutdown gracefully
	rpcChannel := make(chan bool)
	err = sidecar.WithRpcServer(ctx, mds, sm, rc, rpcChannel)
	if err != nil {
		l.Sugar().Fatalw("Failed to start RPC server", zap.Error(err))
	}

	block, err := fetchr.FetchBlock(ctx, 1215893)
	if err != nil {
		l.Sugar().Fatalw("Failed to fetch block", zap.Error(err))
	}

	transactionHash := "0xf6775c38af1d2802bcbc2b7c8959c0d5b48c63a14bfeda0261ba29d76c68c423"
	transaction := &ethereum.EthereumTransaction{}

	for _, tx := range block.Block.Transactions {
		if tx.Hash.Value() == transactionHash {
			transaction = tx
			break
		}
	}

	logIndex := 4
	receipt := block.TxReceipts[transaction.Hash.Value()]
	var interestingLog *ethereum.EthereumEventLog

	for _, log := range receipt.Logs {
		if log.LogIndex.Value() == uint64(logIndex) {
			fmt.Printf("Log: %+v\n", log)
			interestingLog = log
		}
	}

	decodedLog, err := idxr.DecodeLogWithAbi(nil, receipt, interestingLog)
	if err != nil {
		l.Sugar().Fatalw("Failed to decode log", zap.Error(err))
	}
	l.Sugar().Infof("Decoded log: %+v", decodedLog)
}
