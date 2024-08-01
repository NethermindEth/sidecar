package workers

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/Layr-Labs/sidecar/internal/config"
	"github.com/Layr-Labs/sidecar/internal/contractCaller"
	"github.com/Layr-Labs/sidecar/internal/fetcher"
	"github.com/Layr-Labs/sidecar/internal/indexer"
	"github.com/Layr-Labs/sidecar/internal/queue/rabbitmq"
	"github.com/Layr-Labs/sidecar/internal/storage/metadata"
	"github.com/ethereum/go-ethereum/common"
	"github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
	"golang.org/x/xerrors"
)

type RestakedStrategiesIndexerConfig struct {
	QueueName string
	Prefectch int
}

type RestakedStrategiesIndexer struct {
	config         *RestakedStrategiesIndexerConfig
	globalConfig   *config.Config
	logger         *zap.Logger
	indexer        *indexer.Indexer
	fetcher        *fetcher.Fetcher
	rabbitMq       *rabbitmq.RabbitMQ
	contractCaller contractCaller.IContractCaller
}

func NewRestakedStrategiesIndexer(
	rCfg *RestakedStrategiesIndexerConfig,
	idxr *indexer.Indexer,
	f *fetcher.Fetcher,
	l *zap.Logger,
	rmq *rabbitmq.RabbitMQ,
	cfg *config.Config,
	contractCaller contractCaller.IContractCaller,
) *RestakedStrategiesIndexer {
	return &RestakedStrategiesIndexer{
		config:         rCfg,
		logger:         l,
		indexer:        idxr,
		fetcher:        f,
		rabbitMq:       rmq,
		globalConfig:   cfg,
		contractCaller: contractCaller,
	}
}

const DEFAULT_BLOCK_INTERVAL = 3600

func (ci *RestakedStrategiesIndexer) Consume(forever chan struct{}) (*amqp091.Connection, error) {
	conn, err := ci.rabbitMq.Connect()
	if err != nil {
		ci.logger.Sugar().Errorf("Failed to connect to RabbitMQ: %v", err)
		return nil, err
	}
	//defer conn.Close()

	if ci.config.Prefectch > 0 {
		ci.logger.Sugar().Infof("Setting QoS to %d", ci.config.Prefectch)
		if err := ci.rabbitMq.SetQos(ci.config.Prefectch); err != nil {
			ci.logger.Sugar().Errorf("Failed to set QoS: %v", err)
			return conn, err
		}
	}

	ci.logger.Sugar().Infof("Consuming from queue: %s", ci.config.QueueName)
	allBlockMessages, err := ci.rabbitMq.Consume(ci.config.QueueName, "block-indexer", false, false, false, false, nil)
	if err != nil {
		ci.logger.Sugar().Errorf("Failed to register a consumer: %v", err)
		return conn, err
	}

	//var forever chan struct{}
	go func() {
		for d := range allBlockMessages {
			ci.logger.Sugar().Info("Received a message")
			if err := ci.handleMessage(d); err != nil {
				ci.logger.Sugar().Errorf("Failed to handle message: %v", err)
			}
			d.Ack(false)
		}
	}()

	ci.logger.Sugar().Infof(" [*] Waiting for messages. To exit press CTRL+C")
	return conn, nil
}

func (ci *RestakedStrategiesIndexer) handleMessage(message amqp091.Delivery) error {
	ctx := context.Background()

	if ci.config.QueueName == rabbitmq.Queue_restakeStrategies {
		blockProcessedMessage := &rabbitmq.ReIndexRestakedStrategies{}
		if err := json.Unmarshal(message.Body, blockProcessedMessage); err != nil {
			ci.logger.Sugar().Errorf("Failed to unmarshal message: %v", err)
			return err
		}
		return ci.processRestakedStrategiesForBlock(ctx, blockProcessedMessage.BlockNumber)
	}

	if ci.config.QueueName == rabbitmq.Queue_restakeStrategiesAllBlocks {
		blockProcessedMessage := &rabbitmq.BlockProcessedMessage{}
		if err := json.Unmarshal(message.Body, blockProcessedMessage); err != nil {
			ci.logger.Sugar().Errorf("Failed to unmarshal message: %v", err)
			return err
		}
		if blockProcessedMessage.BlockNumber%DEFAULT_BLOCK_INTERVAL != 0 {
			ci.logger.Sugar().Info("Skipping block", zap.Uint64("blockNumber", blockProcessedMessage.BlockNumber))
			return nil
		}
		return ci.processRestakedStrategiesForBlock(ctx, blockProcessedMessage.BlockNumber)
	}

	return xerrors.Errorf("Unknown queue name: %s", ci.config.QueueName)
}

func (ci *RestakedStrategiesIndexer) processRestakedStrategiesForBlock(ctx context.Context, blockNumber uint64) error {
	ci.logger.Sugar().Info(fmt.Sprintf("Processing restaked strategies for block: %v", blockNumber))

	block, err := ci.indexer.MetadataStore.GetBlockByNumber(blockNumber)
	if err != nil {
		ci.logger.Sugar().Errorw(fmt.Sprintf("Failed to fetch block: %v", blockNumber), zap.Error(err))
		return err
	}
	if block == nil {
		ci.logger.Sugar().Errorw(fmt.Sprintf("Block not found: %v", blockNumber))
		return nil
	}

	addresses := make([]string, 0)

	if ci.globalConfig.Environment == config.Environment_PreProd || ci.globalConfig.Environment == config.Environment_Testnet {
		addresses = append(addresses, config.AVSDirectoryAddresses[config.Environment_PreProd][config.Network_Holesky])
		addresses = append(addresses, config.AVSDirectoryAddresses[config.Environment_Testnet][config.Network_Holesky])
	} else {
		addresses = append(addresses, config.AVSDirectoryAddresses[config.Environment_Mainnet][config.Network_Ethereum])
	}

	for _, avsDirectoryAddress := range addresses {
		if err := ci.processRestakedStrategiesForBlockAndAvsDirectory(ctx, block, avsDirectoryAddress); err != nil {
			ci.logger.Sugar().Errorw("Failed to process restaked strategies", zap.Error(err))
		}
	}
	return nil
}

func (ci *RestakedStrategiesIndexer) processRestakedStrategiesForBlockAndAvsDirectory(ctx context.Context, block *metadata.Block, avsDirectoryAddress string) error {
	ci.logger.Sugar().Infof(fmt.Sprintf("Using avs directory address '%s", avsDirectoryAddress))

	blockNumber := block.Number

	avsOperators, err := ci.indexer.MetadataStore.GetLatestActiveAvsOperators(blockNumber, avsDirectoryAddress)
	if err != nil {
		ci.logger.Sugar().Errorw(fmt.Sprintf("Failed to fetch block: %v", blockNumber), zap.Error(err))
		return err
	}

	ci.logger.Sugar().Infow(fmt.Sprintf("Found %d active AVS operators", len(avsOperators)))

	for _, avsOperator := range avsOperators {
		operator := avsOperator.Operator
		avs := avsOperator.Avs

		ci.logger.Sugar().Infow("Fetching restaked strategies for operator",
			zap.String("operator", operator),
			zap.String("avs", avs),
			zap.String("avsDirectoryAddress", avsDirectoryAddress),
			zap.Uint64("blockNumber", blockNumber),
		)
		restakedStrategies, err := ci.contractCaller.GetOperatorRestakedStrategies(ctx, avs, operator, blockNumber)

		if err != nil {
			ci.logger.Sugar().Errorw("Failed to get operator restaked strategies",
				zap.Error(err),
				zap.String("operator", operator),
				zap.String("avs", avs),
				zap.String("avsDirectoryAddress", avsDirectoryAddress),
				zap.Uint64("blockNumber", blockNumber),
			)
			continue
		}
		ci.logger.Sugar().Infow("Fetched restaked strategies for operator: %v",
			zap.Error(err),
			zap.String("operator", operator),
			zap.String("avs", avs),
			zap.String("avsDirectoryAddress", avsDirectoryAddress),
			zap.Uint64("blockNumber", blockNumber),
		)

		for _, restakedStrategy := range restakedStrategies {
			if _, err := ci.indexer.MetadataStore.InsertOperatorRestakedStrategies(avsDirectoryAddress, blockNumber, block.BlockTime, operator, avs, restakedStrategy.String()); err != nil {
				ci.logger.Sugar().Errorw("Failed to save restaked strategy",
					zap.Error(err),
					zap.String("restakedStrategy", restakedStrategy.String()),
					zap.String("operator", operator),
					zap.String("avs", avs),
					zap.String("avsDirectoryAddress", avsDirectoryAddress),
					zap.Uint64("blockNumber", blockNumber),
				)
				continue
			}
			ci.logger.Sugar().Infow("Inserted restaked strategy",
				zap.String("restakedStrategy", restakedStrategy.String()),
				zap.String("operator", operator),
				zap.String("avs", avs),
				zap.String("avsDirectoryAddress", avsDirectoryAddress),
				zap.Uint64("blockNumber", blockNumber),
			)
		}
	}

	return nil
}

func (ci *RestakedStrategiesIndexer) reconcileAddresses(a, b []common.Address) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
