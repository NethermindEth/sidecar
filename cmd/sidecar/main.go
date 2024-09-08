package main

import (
	"context"
	"fmt"
	"github.com/Layr-Labs/sidecar/internal/clients/ethereum"
	"github.com/Layr-Labs/sidecar/internal/clients/etherscan"
	"github.com/Layr-Labs/sidecar/internal/config"
	"github.com/Layr-Labs/sidecar/internal/contractManager"
	"github.com/Layr-Labs/sidecar/internal/contractStore/sqliteContractStore"
	"github.com/Layr-Labs/sidecar/internal/eigenState/stateManager"
	"github.com/Layr-Labs/sidecar/internal/fetcher"
	"github.com/Layr-Labs/sidecar/internal/indexer"
	"github.com/Layr-Labs/sidecar/internal/logger"
	"github.com/Layr-Labs/sidecar/internal/metrics"
	"github.com/Layr-Labs/sidecar/internal/pipeline"
	"github.com/Layr-Labs/sidecar/internal/shutdown"
	"github.com/Layr-Labs/sidecar/internal/sidecar"
	"github.com/Layr-Labs/sidecar/internal/sqlite"
	"github.com/Layr-Labs/sidecar/internal/sqlite/migrations"
	sqliteBlockStore "github.com/Layr-Labs/sidecar/internal/storage/sqlite"
	"go.uber.org/zap"
	"log"
	"time"
)

func main() {
	ctx := context.Background()
	cfg := config.NewConfig()

	fmt.Printf("Config: %+v\n", cfg)

	l, _ := logger.NewLogger(&logger.LoggerConfig{Debug: cfg.Debug})

	sdc, err := metrics.InitStatsdClient(cfg.StatsdUrl)
	if err != nil {
		l.Sugar().Fatal("Failed to setup statsd client", zap.Error(err))
	}

	etherscanClient := etherscan.NewEtherscanClient(cfg, l)
	client := ethereum.NewClient(cfg.EthereumRpcConfig.BaseUrl, l)

	db := sqlite.NewSqlite(cfg.SqliteConfig.GetSqlitePath())

	grm, err := sqlite.NewGormSqliteFromSqlite(db)
	if err != nil {
		l.Error("Failed to create gorm instance", zap.Error(err))
		panic(err)
	}

	migrator := migrations.NewSqliteMigrator(grm, l)
	if err = migrator.MigrateAll(); err != nil {
		log.Fatalf("Failed to migrate: %v", err)
	}

	contractStore := sqliteContractStore.NewSqliteContractStore(grm, l, cfg)
	if err := contractStore.InitializeCoreContracts(); err != nil {
		log.Fatalf("Failed to initialize core contracts: %v", err)
	}

	cm := contractManager.NewContractManager(contractStore, etherscanClient, client, sdc, l)

	mds := sqliteBlockStore.NewSqliteBlockStore(grm, l, cfg)
	if err != nil {
		log.Fatalln(err)
	}

	sm := stateManager.NewEigenStateManager(l, grm)

	fetchr := fetcher.NewFetcher(client, cfg, l)

	idxr := indexer.NewIndexer(mds, contractStore, etherscanClient, cm, client, fetchr, l, cfg)

	p := pipeline.NewPipeline(fetchr, idxr, mds, sm, l)

	// Create new sidecar instance
	sidecar := sidecar.NewSidecar(&sidecar.SidecarConfig{
		GenesisBlockNumber: cfg.GetGenesisBlockNumber(),
	}, cfg, mds, p, l, client)

	// RPC channel to notify the RPC server to shutdown gracefully
	rpcChannel := make(chan bool)
	err = sidecar.WithRpcServer(ctx, mds, sm, rpcChannel)
	if err != nil {
		l.Sugar().Fatalw("Failed to start RPC server", zap.Error(err))
	}

	// Start the sidecar main process in a goroutine so that we can listen for a shutdown signal
	go sidecar.Start(ctx)

	l.Sugar().Info("Started Sidecar")

	gracefulShutdown := shutdown.CreateGracefulShutdownChannel()

	done := make(chan bool)
	shutdown.ListenForShutdown(gracefulShutdown, done, func() {
		l.Sugar().Info("Shutting down...")
		rpcChannel <- true
	}, time.Second*5, l)
}
