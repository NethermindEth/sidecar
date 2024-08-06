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
	"github.com/Layr-Labs/sidecar/internal/queue/rabbitmq"
	"github.com/Layr-Labs/sidecar/internal/shutdown"
	"github.com/Layr-Labs/sidecar/internal/storage/postgresql"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"go.uber.org/zap"
	"log"
	"time"
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

	mds, err := postgresql.NewMetadataStore(grm, l)
	if err != nil {
		log.Fatalln(err)
	}

	fetchr := fetcher.NewFetcher(client, l)

	idxr := indexer.NewIndexer(mds, contractStore, etherscanClient, cm, client, fetchr, l, cfg)

	queues, exchanges := rabbitmq.GetQueuesAndExchanges()
	rmq := rabbitmq.NewRabbitMQ(&rabbitmq.RabbitMQConfig{
		Username:  cfg.RabbitMqConfig.Username,
		Password:  cfg.RabbitMqConfig.Password,
		Url:       cfg.RabbitMqConfig.Url,
		Exchanges: exchanges,
		Queues:    queues,
		Secure:    cfg.RabbitMqConfig.Secure,
	}, l)

	rmqConn, err := rmq.Connect()
	if err != nil {
		l.Sugar().Fatalf("Failed to connect to RabbitMQ", zap.Error(err))
	}
	defer rmqConn.Close()

	shouldShutdown := false
	shouldShutdownCh := make(chan bool)
	gracefulShutdown := shutdown.CreateGracefulShutdownChannel()
	websocketQuitCh := make(chan struct{})

	go shutdown.ListenForShutdown(gracefulShutdown, shouldShutdownCh, func() {
		shouldShutdown = true
		shouldShutdownCh <- true
		websocketQuitCh <- struct{}{}
	}, time.Second*20, l)

	bNumber, err := mds.GetLatestBlock()
	if err != nil {
		l.Sugar().Fatal("Failed to get latest block", zap.Error(err))
	}
	if bNumber == -1 {
		l.Sugar().Fatal("No blocks indexed yet")
	}
	lastIndexedBlockNumber := uint64(bNumber)
	l.Sugar().Infow("Last indexed block", zap.Uint64("blockNumber", lastIndexedBlockNumber))

	onChainBlock, err := client.GetBlockNumber(ctx)
	if err != nil {
		l.Sugar().Fatal("Failed to get block number", zap.Error(err))
	}
	blockNumber, err := hexutil.DecodeUint64(onChainBlock)

	l.Sugar().Infow("On chain block number", zap.Uint64("blockNumber", blockNumber))

	if blockNumber <= lastIndexedBlockNumber {
		l.Sugar().Infow("Got block number is less than last indexed; nothing to do", zap.Uint64("blockNumber", blockNumber), zap.Uint64("lastIndexedBlock", lastIndexedBlockNumber))
		return
	}

	l.Sugar().Info(fmt.Sprintf("Difference: %d\n", blockNumber-(lastIndexedBlockNumber+1)))

	for i := lastIndexedBlockNumber + 1; i <= blockNumber; i++ {
		if shouldShutdown {
			l.Sugar().Infof("ShouldShutdown is set to true, exiting backfill loop...")
			return
		}
		fetchedBlock, indexedBlock, previouslyIndexed, err := idxr.FetchAndIndexBlock(ctx, i, false)
		if err != nil {
			l.Sugar().Errorw("Failed to fetch and index block", zap.Error(err))
			continue
		}
		if previouslyIndexed {
			l.Sugar().Infow("Block already indexed, skipping", zap.Uint64("blockNumber", i))
			continue
		}

		l.Sugar().Infow("Indexed block and transactions", zap.Uint64("blockNumber", i))

		updatedRecentBlock, err := client.GetBlockNumber(ctx)
		if err != nil {
			l.Sugar().Fatalw("Failed to get block number", zap.Error(err))
		}

		updatedRecentBlockNumber, err := hexutil.DecodeUint64(updatedRecentBlock)
		if updatedRecentBlockNumber > blockNumber {
			l.Sugar().Infow("Block number has increased", zap.Uint64("blockNumber", updatedRecentBlockNumber))
			blockNumber = updatedRecentBlockNumber
		}
	}

	if shouldShutdown {
		l.Sugar().Infof("ShouldShutdown is set to true, cleaning up...")
		return
	}

	wsc, err := client.GetWebsocketConnection(cfg.EthereumRpcConfig.WsUrl)
	if err != nil {
		l.Sugar().Fatalf("Failed to get websocket connection", zap.Error(err))
	}
	defer wsc.Close()

	client.ListenForNewBlocks(context.Background(), wsc, websocketQuitCh, func(header *types.Header) error {
		if shouldShutdown {
			l.Sugar().Infof("ShouldShutdown is set to true, exiting websocket block listener...")
			return nil
		}
		l.Sugar().Infow("Got block number", zap.Uint64("blockNumber", header.Number.Uint64()))
		// Delay by 2 blocks to ensure finality and not have to deal with re-orgs
		blockNumber := header.Number.Uint64() - 2

		block, err := mds.GetBlockByNumber(blockNumber)
		if err != nil {
			l.Sugar().Errorw("Failed to get block by number", zap.Error(err))
			return err
		}
		if block != nil {
			l.Sugar().Info("Block already indexed", zap.Uint64("blockNumber", blockNumber))
			return nil
		}

		_, _, _, err = idxr.FetchAndIndexBlock(ctx, blockNumber, false)
		if err != nil {
			l.Sugar().Errorw("Failed to fetch and index block", zap.Error(err))
			return err
		}

		return nil
	})
}
