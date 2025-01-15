package types

import "github.com/Layr-Labs/sidecar/pkg/storage"

type IMetaStateModel interface {
	TableName() string

	SetupStateForBlock(blockNumber uint64) error

	CleanupStateForBlock(blockNumber uint64) error

	IsInterestingLog(log *storage.TransactionLog) bool

	HandleTransactionLog(log *storage.TransactionLog) (interface{}, error)

	CommitFinalState(blockNumber uint64) error

	DeleteState(startBlockNumber uint64, endBlockNumber uint64) error
}
