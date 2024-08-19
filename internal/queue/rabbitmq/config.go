package rabbitmq

import "github.com/Layr-Labs/sidecar/internal/clients/ethereum"

const (
	Queue_blockIndexer               = "block-indexer"
	Queue_transactionParser          = "transaction-parser"
	Queue_contractIndexer            = "contract-indexer"
	Queue_restakeStrategiesAllBlocks = "restake-strategies-all-blocks"
	Queue_restakeStrategies          = "restake-strategies"

	RoutingKey_blockIndexer      = "block-indexer"
	RoutingKey_transactionParser = "transaction-parser"
	RoutingKey_contractIndexer   = "contract-indexer"
	RoutingKey_restakeStrategies = "restake-strategies"

	Exchange_blocklake = "blocklake"
	Exchange_blocks    = "blocks"
)

func GetQueuesAndExchanges() ([]*RabbitMQQueue, []*RabbitMQExchange) {
	queues := []*RabbitMQQueue{
		{
			Name:           Queue_blockIndexer,
			Durable:        true,
			AutoAck:        false,
			Exclusive:      false,
			BindExchange:   Exchange_blocklake,
			BindRoutingKey: RoutingKey_blockIndexer,
		}, {
			Name:           Queue_transactionParser,
			Durable:        true,
			AutoAck:        false,
			Exclusive:      false,
			BindExchange:   Exchange_blocklake,
			BindRoutingKey: RoutingKey_transactionParser,
		}, {
			Name:           Queue_contractIndexer,
			Durable:        true,
			AutoAck:        false,
			Exclusive:      false,
			BindExchange:   Exchange_blocklake,
			BindRoutingKey: RoutingKey_contractIndexer,
		},
		// We bind Queue_restakeStrategiesAllBlocks to two different exchanges:
		// - blocklake is used for directly queueing to it
		// - blocks is a fanout exchange for anything that wants to listen to all blocks produced.
		{
			Name:           Queue_restakeStrategies,
			Durable:        true,
			AutoAck:        false,
			Exclusive:      false,
			BindExchange:   Exchange_blocklake,
			BindRoutingKey: RoutingKey_restakeStrategies,
		}, {
			Name:           Queue_restakeStrategiesAllBlocks,
			Durable:        true,
			AutoAck:        false,
			Exclusive:      false,
			BindExchange:   Exchange_blocks,
			BindRoutingKey: RoutingKey_restakeStrategies,
		},
	}

	exchanges := []*RabbitMQExchange{
		{
			Name:       "blocklake",
			Durable:    true,
			AutoDelete: false,
			Kind:       "topic",
		}, {
			Name:       "blocks",
			Durable:    true,
			AutoDelete: false,
			Kind:       "fanout",
		},
	}

	return queues, exchanges
}

type BlockIndexerMessage struct {
	BlockNumber uint64
	Reindex     bool
}

type TransactionParserMessage struct {
	BlockNumber     uint64
	BlockSequenceId uint64
	Transactions    []*ethereum.EthereumTransaction
	Receipts        map[string]*ethereum.EthereumTransactionReceipt
	Reprocess       bool
}

type ContractIndexerParserMessage struct {
	BlockNumber uint64
}

type ReIndexTransactionMessage struct {
	TransactionHash string
}

type BlockProcessedMessage struct {
	BlockNumber uint64
}

type ReIndexRestakedStrategies struct {
	BlockNumber uint64
}
