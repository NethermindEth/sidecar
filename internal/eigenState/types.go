package eigenState

import (
	"github.com/Layr-Labs/sidecar/internal/storage"
)

type StateRoot string

type IEigenStateModel interface {
	// Determine if the log is interesting to the state model
	IsInterestingLog(log *storage.TransactionLog) bool

	// Allow the state model to handle the state change
	//
	// Returns the saved value. Listed as an interface because go generics suck
	HandleStateChange(log *storage.TransactionLog) (interface{}, error)

	// Once all state changes are processed, calculate and write final state
	WriteFinalState(blockNumber uint64) error

	// Generate the state root for the model
	GenerateStateRoot(blockNumber uint64) (StateRoot, error)
}

// Map of block number to function that will transition the state to the next block
type StateTransitions[T interface{}] map[uint64]func(log *storage.TransactionLog) (*T, error)
