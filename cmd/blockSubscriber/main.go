package main

import (
	"context"
	"encoding/json"
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
	"github.com/Layr-Labs/sidecar/internal/queue/rabbitmq"
	"github.com/Layr-Labs/sidecar/internal/storage/postgresql"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
	"log"
)

func publishBlockProcessedToQueue(blockNumber uint64, rmq *rabbitmq.RabbitMQ, l *zap.Logger) error {
	blockMessage := &rabbitmq.BlockProcessedMessage{
		BlockNumber: blockNumber,
	}
	msgJson, err := json.Marshal(blockMessage)
	if err != nil {
		l.Sugar().Errorw("Failed to marshal message", zap.Error(err))
		return err
	}
	rmq.Publish(rabbitmq.Exchange_blocks, rabbitmq.RoutingKey_blockIndexer, amqp091.Publishing{
		ContentType: "application/json",
		Body:        msgJson,
	})
	return nil
}

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

	mds, err := postgresql.NewMetadataStore(grm, l)
	if err != nil {
		log.Fatalln(err)
	}

	fetchr := fetcher.NewFetcher(client, l)

	idxr := indexer.NewIndexer(mds, contractStore, etherscanClient, cm, client, fetchr, l)

	lastIndexedBlockNumber, err := mds.GetLatestBlock()
	if err != nil {
		l.Sugar().Fatal("Failed to get latest block", zap.Error(err))
	}
	l.Sugar().Infow("Last indexed block", zap.Uint64("block", lastIndexedBlockNumber))

	onChainBlock, err := client.GetBlockNumber(ctx)
	if err != nil {
		l.Sugar().Fatal("Failed to get block number", zap.Error(err))
	}
	l.Sugar().Infow("On chain block number", zap.String("block", onChainBlock))

	blockNumber, err := hexutil.DecodeUint64(onChainBlock)

	if blockNumber <= lastIndexedBlockNumber {
		l.Sugar().Infow("Got block number is less than last indexed; nothing to do", zap.Uint64("block", blockNumber), zap.Uint64("lastIndexedBlock", lastIndexedBlockNumber))
		return
	}

	l.Sugar().Info(fmt.Sprintf("Difference: %d\n", blockNumber-(lastIndexedBlockNumber+1)))

	for i := lastIndexedBlockNumber + 1; i <= blockNumber; i++ {
		fetchedBlock, indexedBlock, _, err := idxr.FetchAndIndexBlock(ctx, i, false)
		if err != nil {
			l.Sugar().Errorw("Failed to fetch and index block", zap.Error(err))
			continue
		}

		idxr.ParseAndIndexTransactionLogs(ctx, fetchedBlock, indexedBlock.Id)
		l.Sugar().Infow("Indexed block and transactions", zap.String("block", onChainBlock))

		updatedRecentBlock, err := client.GetBlockNumber(ctx)
		if err != nil {
			l.Sugar().Fatalw("Failed to get block number", zap.Error(err))
		}

		updatedRecentBlockNumber, err := hexutil.DecodeUint64(updatedRecentBlock)
		if updatedRecentBlockNumber > blockNumber {
			l.Sugar().Infow("Block number has increased", zap.Uint64("block", updatedRecentBlockNumber))
			blockNumber = updatedRecentBlockNumber
		}
	}

	wsc, err := client.GetWebsocketConnection(cfg.EthereumRpcConfig.WsUrl)
	if err != nil {
		l.Sugar().Fatalf("Failed to get websocket connection", zap.Error(err))
	}
	defer wsc.Close()

	quitCh := make(chan struct{})

	client.ListenForNewBlocks(context.Background(), wsc, quitCh, func(header *types.Header) error {
		l.Sugar().Infow("Got block number", zap.Uint64("block", header.Number.Uint64()))
		// Delay by 2 blocks to ensure finality and not have to deal with re-orgs
		blockNumber := header.Number.Uint64() - 2

		block, err := mds.GetBlockByNumber(blockNumber)
		if err != nil {
			l.Sugar().Errorw("Failed to get block by number", zap.Error(err))
			return err
		}
		if block != nil {
			l.Sugar().Info("Block already indexed", zap.Uint64("block", blockNumber))
			return nil
		}

		fetchedBlock, indexedBlock, _, err := idxr.FetchAndIndexBlock(ctx, blockNumber, false)
		if err != nil {
			l.Sugar().Errorw("Failed to fetch and index block", zap.Error(err))
			return err
		}

		idxr.ParseAndIndexTransactionLogs(ctx, fetchedBlock, indexedBlock.Id)
		return nil
	})
}
