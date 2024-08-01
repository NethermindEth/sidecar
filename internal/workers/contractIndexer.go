package workers

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/Layr-Labs/sidecar/internal/fetcher"
	"github.com/Layr-Labs/sidecar/internal/indexer"
	"github.com/Layr-Labs/sidecar/internal/queue/rabbitmq"
	"github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

type ContractIndexerConfig struct {
	QueueName string
	Prefectch int
}

type ContractIndexer struct {
	config   *ContractIndexerConfig
	logger   *zap.Logger
	indexer  *indexer.Indexer
	fetcher  *fetcher.Fetcher
	rabbitMq *rabbitmq.RabbitMQ
}

func NewContractIndexer(
	cfg *ContractIndexerConfig,
	idxr *indexer.Indexer,
	f *fetcher.Fetcher,
	l *zap.Logger,
	rmq *rabbitmq.RabbitMQ,
) *ContractIndexer {
	return &ContractIndexer{
		config:   cfg,
		logger:   l,
		indexer:  idxr,
		fetcher:  f,
		rabbitMq: rmq,
	}
}

func (ci *ContractIndexer) Consume() error {
	conn, err := ci.rabbitMq.Connect()
	if err != nil {
		ci.logger.Sugar().Errorf("Failed to connect to RabbitMQ: %v", err)
		return err
	}
	defer conn.Close()

	if ci.config.Prefectch > 0 {
		ci.logger.Sugar().Infof("Setting QoS to %d", ci.config.Prefectch)
		if err := ci.rabbitMq.SetQos(ci.config.Prefectch); err != nil {
			ci.logger.Sugar().Errorf("Failed to set QoS: %v", err)
			return err
		}
	}

	ci.logger.Sugar().Infof("Consuming from queue: %s", ci.config.QueueName)
	msgs, err := ci.rabbitMq.Consume(ci.config.QueueName, "block-indexer", false, false, false, false, nil)
	if err != nil {
		ci.logger.Sugar().Errorf("Failed to register a consumer: %v", err)
		return err
	}

	var forever chan struct{}

	go func() {
		for d := range msgs {
			ci.logger.Sugar().Debugw("Received a message")
			if err := ci.handleMessage(d); err != nil {
				ci.logger.Sugar().Errorf("Failed to handle message: %v", err)
			}
			d.Ack(false)
		}
	}()

	ci.logger.Sugar().Infof(" [*] Waiting for messages. To exit press CTRL+C")
	<-forever
	return nil
}

func (ci *ContractIndexer) handleMessage(message amqp091.Delivery) error {
	ctx := context.Background()
	contractIndexerMessage := &rabbitmq.ContractIndexerParserMessage{}
	if err := json.Unmarshal(message.Body, contractIndexerMessage); err != nil {
		ci.logger.Sugar().Errorf("Failed to unmarshal message: %v", err)
		return err
	}

	return ci.processContractsForBlock(ctx, contractIndexerMessage)
}

func (ci *ContractIndexer) processContractsForBlock(ctx context.Context, contractIndexerMessage *rabbitmq.ContractIndexerParserMessage) error {
	blockNumber := contractIndexerMessage.BlockNumber

	ci.logger.Sugar().Info(fmt.Sprintf("Processing contracts for block: %v", blockNumber))

	fetchedBlock, err := ci.indexer.Fetcher.FetchBlock(ctx, blockNumber)
	if err != nil {
		ci.logger.Sugar().Errorw(fmt.Sprintf("Failed to fetch block: %v", blockNumber), zap.Error(err))
		return err
	}

	block, err := ci.indexer.MetadataStore.GetBlockByNumber(blockNumber)
	if err != nil {
		ci.logger.Sugar().Errorw(fmt.Sprintf("Failed to get block: %v", blockNumber), zap.Error(err))
		return err
	}
	if block == nil {
		ci.logger.Sugar().Errorw(fmt.Sprintf("Block not found: %v", blockNumber))
		return fmt.Errorf("block not found")
	}

	ci.indexer.IndexContractsForBlock(block, fetchedBlock, true)
	return nil
}
