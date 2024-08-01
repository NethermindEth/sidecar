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

type TransactionIndexerConfig struct {
	QueueName string
	Prefectch int
}

type TransactionIndexer struct {
	config   *TransactionIndexerConfig
	logger   *zap.Logger
	indexer  *indexer.Indexer
	fetcher  *fetcher.Fetcher
	rabbitMq *rabbitmq.RabbitMQ
}

func NewTransactionIndexer(
	cfg *TransactionIndexerConfig,
	idxr *indexer.Indexer,
	f *fetcher.Fetcher,
	l *zap.Logger,
	rmq *rabbitmq.RabbitMQ,
) *TransactionIndexer {
	return &TransactionIndexer{
		config:   cfg,
		logger:   l,
		indexer:  idxr,
		fetcher:  f,
		rabbitMq: rmq,
	}
}

func (bi *TransactionIndexer) Consume() error {
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
			bi.logger.Sugar().Infof("Received a message")
			if err := bi.handleMessage(d); err != nil {
				bi.logger.Sugar().Errorf("Failed to handle message: %v", err)
			}
			d.Ack(false)
		}
	}()

	bi.logger.Sugar().Infof(" [*] Waiting for messages. To exit press CTRL+C")
	<-forever
	return nil
}

func (bi *TransactionIndexer) handleMessage(message amqp091.Delivery) error {
	ctx := context.Background()
	txIndexerMessage := &rabbitmq.TransactionParserMessage{}
	if err := json.Unmarshal(message.Body, txIndexerMessage); err != nil {
		bi.logger.Sugar().Errorf("Failed to unmarshal message: %v", err)
		return err
	}

	if txIndexerMessage.Reprocess {
		bi.logger.Sugar().Infow(fmt.Sprintf("Reprocessing logs for block '%d'", txIndexerMessage.BlockNumber),
			zap.Uint64("block", txIndexerMessage.BlockNumber),
		)
		return bi.reprocessLogsForBlock(ctx, txIndexerMessage.BlockNumber)
	}
	return bi.processLogsForBlock(ctx, txIndexerMessage)

	return nil
}

func (bi *TransactionIndexer) processLogsForBlock(ctx context.Context, txIndexerMessage *rabbitmq.TransactionParserMessage) error {
	for i, tx := range txIndexerMessage.Transactions {
		txReceipt, ok := txIndexerMessage.Receipts[tx.Hash.Value()]
		if !ok {
			bi.logger.Sugar().Errorw("failed to get transaction receipt", zap.String("txHash", tx.Hash.Value()))
			continue
		}

		parsedTransactionLogs, err := bi.indexer.ParseTransactionLogs(tx, txReceipt)
		if err != nil {
			bi.logger.Sugar().Errorw("failed to process transaction logs",
				zap.Error(err),
				zap.String("txHash", tx.Hash.Value()),
				zap.Uint64("block", tx.BlockNumber.Value()),
			)
			continue
		}
		if parsedTransactionLogs == nil {
			bi.logger.Sugar().Debugw("Log line is nil",
				zap.String("txHash", tx.Hash.Value()),
				zap.Uint64("block", tx.BlockNumber.Value()),
				zap.Int("logIndex", i),
			)
			continue
		}

		for _, log := range parsedTransactionLogs.Logs {
			_, err := bi.indexer.IndexLog(ctx, txIndexerMessage.BlockNumber, txIndexerMessage.BlockSequenceId, tx.Hash.Value(), tx.Index.Value(), log)
			if err != nil {
				bi.logger.Sugar().Errorw("failed to index log",
					zap.Error(err),
					zap.String("txHash", tx.Hash.Value()),
					zap.Uint64("block", tx.BlockNumber.Value()),
				)
			}
		}

		upgradedLogs := bi.indexer.FindContractUpgradedLogs(parsedTransactionLogs.Logs)
		if len(upgradedLogs) > 0 {
			bi.logger.Sugar().Debugw("Found contract upgrade logs",
				zap.String("txHash", tx.Hash.Value()),
				zap.Uint64("block", tx.BlockNumber.Value()),
				zap.Int("count", len(upgradedLogs)),
			)

			bi.indexer.IndexContractUpgrades(txIndexerMessage.BlockNumber, upgradedLogs, false)
		}
	}
	return nil
}

func (bi *TransactionIndexer) reprocessLogsForBlock(ctx context.Context, blockNumber uint64) error {
	fetchedBlock, err := bi.fetcher.FetchBlock(ctx, blockNumber)
	if err != nil {
		bi.logger.Sugar().Errorw("failed to fetch block",
			zap.Error(err),
			zap.Uint64("blockNumber", blockNumber),
		)
		return err
	}

	block, err := bi.indexer.MetadataStore.GetBlockByNumber(blockNumber)
	if err != nil {
		bi.logger.Sugar().Errorw("failed to get block by number from metadata store",
			zap.Error(err),
			zap.Uint64("blockNumber", blockNumber),
		)
		return err
	}
	if block == nil {
		bi.logger.Sugar().Debugw("Block was not indexed previously, skipping",
			zap.Uint64("blockNumber", blockNumber),
		)
		return nil
	}

	err = bi.indexer.MetadataStore.DeleteTransactionLogsForBlock(blockNumber)
	if err != nil {
		bi.logger.Sugar().Errorw("failed to delete transaction logs",
			zap.Error(err),
			zap.Uint64("blockNumber", blockNumber),
		)
		return nil
	} else {
		bi.logger.Sugar().Debugw("Deleted transaction logs",
			zap.Uint64("blockNumber", blockNumber),
		)

	}

	bi.logger.Sugar().Debugf("Reprocessing %v logs for block", len(fetchedBlock.Block.Transactions))
	fmt.Printf("Transactions: %+v\n", fetchedBlock.TxReceipts)
	err = bi.processLogsForBlock(ctx, &rabbitmq.TransactionParserMessage{
		BlockNumber:     fetchedBlock.Block.Number.Value(),
		BlockSequenceId: block.Id,
		Transactions:    fetchedBlock.Block.Transactions,
		Receipts:        fetchedBlock.TxReceipts,
	})
	if err != nil {
		bi.logger.Sugar().Errorw("failed to process logs for block",
			zap.Error(err),
			zap.Uint64("blockNumber", blockNumber),
		)
		return err
	}
	return err
}
