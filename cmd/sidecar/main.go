package main

import (
	"context"
	"fmt"
	"github.com/Layr-Labs/sidecar/internal/clients/ethereum"
	"github.com/Layr-Labs/sidecar/internal/clients/etherscan"
	"github.com/Layr-Labs/sidecar/internal/config"
	"github.com/Layr-Labs/sidecar/internal/contractManager"
	"github.com/Layr-Labs/sidecar/internal/contractStore/sqliteContractStore"
	"github.com/Layr-Labs/sidecar/internal/fetcher"
	"github.com/Layr-Labs/sidecar/internal/indexer"
	"github.com/Layr-Labs/sidecar/internal/logger"
	"github.com/Layr-Labs/sidecar/internal/metrics"
	"github.com/Layr-Labs/sidecar/internal/pipeline"
	"github.com/Layr-Labs/sidecar/internal/sidecar"
	"github.com/Layr-Labs/sidecar/internal/sqlite"
	"github.com/Layr-Labs/sidecar/internal/sqlite/migrations"
	"github.com/Layr-Labs/sidecar/internal/storage/postgresql"
	"go.uber.org/zap"
	"log"
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
	migrator.MigrateAll()
	if err = migrator.MigrateAll(); err != nil {
		log.Fatalf("Failed to migrate: %v", err)
	}

	contractStore := sqliteContractStore.NewSqliteContractStore(grm, l)

	cm := contractManager.NewContractManager(contractStore, etherscanClient, client, sdc, l)

	mds, err := postgresql.NewPostgresBlockStore(grm, cfg, l)
	if err != nil {
		log.Fatalln(err)
	}

	fetchr := fetcher.NewFetcher(client, cfg, l)

	idxr := indexer.NewIndexer(mds, contractStore, etherscanClient, cm, client, fetchr, l, cfg)

	p := pipeline.NewPipeline(fetchr, idxr, mds, l)

	sidecar := sidecar.NewSidecar(&sidecar.SidecarConfig{
		GenesisBlockNumber: cfg.GetGenesisBlockNumber(),
	}, cfg, mds, p, l, client)

	sidecar.Start(ctx)
}
