package workers

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/DataDog/datadog-go/v5/statsd"
	"github.com/Layr-Labs/sidecar/internal/fetcher"
	"github.com/Layr-Labs/sidecar/internal/indexer"
	"github.com/Layr-Labs/sidecar/internal/metrics"
	"github.com/Layr-Labs/sidecar/internal/queue/rabbitmq"
	"github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

type BlockIndexerConfig struct {
	QueueName string
	Prefectch int
}

type BlockIndexer struct {
	config   *BlockIndexerConfig
	logger   *zap.Logger
	indexer  *indexer.Indexer
	fetcher  *fetcher.Fetcher
	rabbitMq *rabbitmq.RabbitMQ
	statsd   *statsd.Client
}

func NewBlockIndexer(
	cfg *BlockIndexerConfig,
	idxr *indexer.Indexer,
	f *fetcher.Fetcher,
	l *zap.Logger,
	rmq *rabbitmq.RabbitMQ,
) *BlockIndexer {
	s, _ := metrics.InitStatsdClient("")
	return &BlockIndexer{
		config:   cfg,
		logger:   l,
		indexer:  idxr,
		fetcher:  f,
		rabbitMq: rmq,
		statsd:   s,
	}
}

func (bi *BlockIndexer) Consume() error {
	conn, err := bi.rabbitMq.Connect()
	if err != nil {
		bi.logger.Sugar().Errorf("Failed to connect to RabbitMQ: %v", err)
		return err
	}
	defer conn.Close()

	if bi.config.Prefectch > 0 {
		bi.logger.Sugar().Infof("Setting QoS to %d", bi.config.Prefectch)
		if err := bi.rabbitMq.SetQos(bi.config.Prefectch); err != nil {
			bi.logger.Sugar().Errorf("Failed to set QoS: %v", err)
			return err
		}
	}

	msgs, err := bi.rabbitMq.Consume(bi.config.QueueName, "block-indexer", false, false, false, false, nil)
	if err != nil {
		bi.logger.Sugar().Errorf("Failed to register a consumer: %v", err)
		return err
	}

	var forever chan struct{}

	go func() {
		for d := range msgs {
			bi.logger.Sugar().Infof("Received a message: %s", d.Body)
			bi.statsd.Incr(Metric_blockMessagesReceived, nil, 1)

			blockIndexerMessage := &rabbitmq.BlockIndexerMessage{}
			if err := json.Unmarshal(d.Body, blockIndexerMessage); err != nil {
				bi.logger.Sugar().Errorf("Failed to unmarshal message: %v", err)
				d.Nack(false, false)
				return
			}

			if err := bi.handleMessage(blockIndexerMessage); err != nil {
				bi.logger.Sugar().Errorf("Failed to handle message: %v", err)
			}
			d.Ack(false)
		}
	}()

	bi.logger.Sugar().Infow(" [*] Waiting for messages. To exit press CTRL+C")
	<-forever
	return nil
}

const (
	Metric_blockMessagesReceived = "block_messages_received"
	Metric_blockSuccesses        = "block_successes"
	Metric_blockFailures         = "block_failures"
	Metric_transactionSuccesses  = "transaction_successes"
	Metric_transactionFailures   = "transaction_failures"
)

func (bi *BlockIndexer) handleMessage(blockIndexerMessage *rabbitmq.BlockIndexerMessage) error {
	ctx := context.Background()

	fetchedBlock, indexedBlock, previouslyIndexed, err := bi.indexer.FetchAndIndexBlock(ctx, blockIndexerMessage.BlockNumber, blockIndexerMessage.Reindex)

	if err != nil {
		bi.statsd.Incr(Metric_blockFailures, nil, 1)
		bi.logger.Sugar().Errorw(fmt.Sprintf("Failed to fetch and index block: %v", blockIndexerMessage.BlockNumber), zap.Error(err))
	} else {
		bi.statsd.Incr(Metric_blockSuccesses, nil, 1)
	}

	transactionParserMessage := &rabbitmq.TransactionParserMessage{
		BlockNumber:     blockIndexerMessage.BlockNumber,
		BlockSequenceId: indexedBlock.Id,
		Transactions:    fetchedBlock.Block.Transactions,
		Receipts:        fetchedBlock.TxReceipts,
	}
	if blockIndexerMessage.Reindex && previouslyIndexed {
		blockIndexerMessage.Reindex = true
	}

	msgJson, err := json.Marshal(transactionParserMessage)
	if err != nil {
		bi.logger.Sugar().Errorw("Failed to marshal message", zap.Error(err))
		return err
	}
	bi.rabbitMq.Publish(rabbitmq.Exchange_blocklake, rabbitmq.RoutingKey_transactionParser, amqp091.Publishing{
		ContentType: "application/json",
		Body:        msgJson,
	})

	if blockIndexerMessage.Reindex == false || previouslyIndexed == false {
		// Dont notify all on a reindex, but do always notify if the block wasnt previously indexed
		blockMessage := &rabbitmq.BlockProcessedMessage{
			BlockNumber: blockIndexerMessage.BlockNumber,
		}
		msgJson, err = json.Marshal(blockMessage)
		if err != nil {
			bi.logger.Sugar().Errorw("Failed to marshal message", zap.Error(err))
			return err
		}
		bi.rabbitMq.Publish(rabbitmq.Exchange_blocks, rabbitmq.RoutingKey_blockIndexer, amqp091.Publishing{
			ContentType: "application/json",
			Body:        msgJson,
		})
	}

	return nil
}
