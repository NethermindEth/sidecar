package types

import "github.com/Layr-Labs/sidecar/pkg/storage"

type IMetaStateModel interface {
	ModelName() string

	SetupStateForBlock(blockNumber uint64) error

	CleanupProcessedStateForBlock(blockNumber uint64) error

	IsInterestingLog(log *storage.TransactionLog) bool

	HandleTransactionLog(log *storage.TransactionLog) (interface{}, error)

	CommitFinalState(blockNumber uint64) ([]interface{}, error)

	DeleteState(startBlockNumber uint64, endBlockNumber uint64) error
}
