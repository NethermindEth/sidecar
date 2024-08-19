package main

import (
	"context"
	"fmt"
	"github.com/Layr-Labs/sidecar/internal/clients/ethereum"
	"github.com/Layr-Labs/sidecar/internal/clients/etherscan"
	"github.com/Layr-Labs/sidecar/internal/config"
	"github.com/Layr-Labs/sidecar/internal/contractManager"
	"github.com/Layr-Labs/sidecar/internal/contractStore/pgContractStore"
	"github.com/Layr-Labs/sidecar/internal/fetcher"
	"github.com/Layr-Labs/sidecar/internal/indexer"
	"github.com/Layr-Labs/sidecar/internal/logger"
	"github.com/Layr-Labs/sidecar/internal/metrics"
	"github.com/Layr-Labs/sidecar/internal/postgres"
	"github.com/Layr-Labs/sidecar/internal/postgres/migrations"
	"github.com/Layr-Labs/sidecar/internal/storage/postgresql"
	"go.uber.org/zap"
	"log"
	"strings"
)

func main() {
	ctx := context.Background()
	cfg := config.NewConfig()

	l, _ := logger.NewLogger(&logger.LoggerConfig{Debug: cfg.Debug})

	sdc, err := metrics.InitStatsdClient(cfg.StatsdUrl)
	if err != nil {
		l.Sugar().Fatal("Failed to setup statsd client", zap.Error(err))
	}

	etherscanClient := etherscan.NewEtherscanClient(cfg, l)
	client := ethereum.NewClient(cfg.EthereumRpcConfig.BaseUrl, l)

	db, err := postgres.NewPostgres(&postgres.PostgresConfig{
		Host:     cfg.PostgresConfig.Host,
		Port:     cfg.PostgresConfig.Port,
		Username: cfg.PostgresConfig.Username,
		Password: cfg.PostgresConfig.Password,
		DbName:   cfg.PostgresConfig.DbName,
	})
	if err != nil {
		l.Error("Failed to setup postgres connection", zap.Error(err))
		panic(err)
	}

	grm, err := postgres.NewGormFromPostgresConnection(db.Db)
	if err != nil {
		l.Error("Failed to create gorm instance", zap.Error(err))
		panic(err)
	}

	migrator := migrations.NewMigrator(db.Db, grm, l)
	migrator.MigrateAll()
	if err = migrator.MigrateAll(); err != nil {
		log.Fatalf("Failed to migrate: %v", err)
	}

	contractStore, err := pgContractStore.NewPgContractStore(grm, l)

	cm := contractManager.NewContractManager(contractStore, etherscanClient, client, sdc, l)

	mds, err := postgresql.NewPostgresBlockStore(grm, l)
	if err != nil {
		log.Fatalln(err)
	}

	fetchr := fetcher.NewFetcher(client, l)

	idxr := indexer.NewIndexer(mds, contractStore, etherscanClient, cm, client, fetchr, l)
	fmt.Printf("Indexer: %+v\n", idxr)

	//------------------------- DEBUG STUFF HERE ---------------------------

	transactionHash := strings.ToLower("0xb6271a924f7e9d46678b86e7a37dbced1af72f84a6c6136d70941a1e686dfddb")

	tx, err := client.GetTransactionByHash(ctx, transactionHash)
	if err != nil {
		l.Sugar().Fatal(err)
	}
	fmt.Printf("Tranasction: %+v\n", tx)

	blockNumber := uint64(1641545)

	b, err := fetchr.FetchBlock(ctx, blockNumber)
	if err != nil {
		l.Sugar().Errorw(fmt.Sprintf("Failed to fetch block: %v", blockNumber), zap.Error(err))
		return
	}

	var txToParse *ethereum.EthereumTransaction
	for _, t := range b.Block.Transactions {
		if strings.ToLower(t.Hash.Value()) == transactionHash {
			txToParse = t
			break
		}
	}
	if txToParse == nil {
		l.Sugar().Fatal(fmt.Sprintf("Failed to find transaction: %v", transactionHash))
	}
	for _, log := range b.TxReceipts[txToParse.Hash.Value()].Logs {
		fmt.Printf("Log: %+v\n", log)
	}
	parsed, err := idxr.ParseTransactionLogs(txToParse, b.TxReceipts[txToParse.Hash.Value()])
	if err != nil {
		l.Sugar().Fatal("Failed to parse transaction logs", zap.Error(err))
		return
	}

	for _, log := range parsed.Logs {
		fmt.Printf("Log: %+v\n", log)
	}

	// upgradedLogs := idxr.FindContractUpgradedLogs(parsed.Logs)
	//
	// fmt.Printf("-----\n")
	// for _, log := range upgradedLogs {
	// 	fmt.Printf("Log: %+v\n", log)
	// }
	//
	// idxr.IndexContractUpgrades(blockNumber, upgradedLogs)
	//
	// tree, err := contractStore.GetContractWithProxyContract("0xb22ef643e1e067c994019a4c19e403253c05c2b0", 1628382)
	// if err != nil {
	// 	l.Sugar().Errorw("Failed to get contract tree", zap.Error(err))
	// 	return
	// }
	// fmt.Printf("Tree: %+v\n", tree)

}
