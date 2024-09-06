package types

import (
	"github.com/Layr-Labs/sidecar/internal/storage"
)

type StateRoot string

type IEigenStateModel interface {
	// GetModelName
	// Get the name of the model
	GetModelName() string

	// IsInterestingLog
	//Determine if the log is interesting to the state model
	IsInterestingLog(log *storage.TransactionLog) bool

	// StartBlockProcessing
	// Perform any necessary setup for processing a block
	StartBlockProcessing(blockNumber uint64) error

	// HandleStateChange
	// Allow the state model to handle the state change
	//
	// Returns the saved value. Listed as an interface because go generics suck
	HandleStateChange(log *storage.TransactionLog) (interface{}, error)

	// CommitFinalState
	// Once all state changes are processed, commit the final state to the database
	CommitFinalState(blockNumber uint64) error

	// GenerateStateRoot
	// Generate the state root for the model
	GenerateStateRoot(blockNumber uint64) (StateRoot, error)

	// ClearAccumulatedState
	// Clear the accumulated state for the model to free up memory
	ClearAccumulatedState(blockNumber uint64) error
}

// StateTransitions
// Map of block number to function that will transition the state to the next block
type StateTransitions[T interface{}] map[uint64]func(log *storage.TransactionLog) (*T, error)
