package eigenState

import (
	"github.com/Layr-Labs/sidecar/internal/storage"
	"go.uber.org/zap"
)

type EigenStateManager struct {
	StateModels []IEigenStateModel
	logger      *zap.Logger
}

func NewEigenStateManager(logger *zap.Logger) *EigenStateManager {
	return &EigenStateManager{
		StateModels: make([]IEigenStateModel, 0),
		logger:      logger,
	}
}

// Allows a model to register itself with the state manager
func (e *EigenStateManager) RegisterState(state IEigenStateModel) {
	e.StateModels = append(e.StateModels, state)
}

// Given a log, allow each state model to determine if/how to process it
func (e *EigenStateManager) HandleLogStateChange(log *storage.TransactionLog) error {
	for _, state := range e.StateModels {
		if state.IsInterestingLog(log) {
			_, err := state.HandleStateChange(log)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// With all transactions/logs processed for a block, commit the final state to the table
func (e *EigenStateManager) CommitFinalState(blockNumber uint64) error {
	for _, state := range e.StateModels {
		err := state.WriteFinalState(blockNumber)
		if err != nil {
			return err
		}
	}
	return nil
}

type StateRoot string

func (e *EigenStateManager) GenerateStateRoot(blockNumber uint64) (StateRoot, error) {
	roots := make([]StateRoot, len(e.StateModels))
	for i, state := range e.StateModels {
		root, err := state.GenerateStateRoot(blockNumber)
		if err != nil {
			return "", err
		}
		roots[i] = root
	}
	// TODO: generate this
	return "", nil
}

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

type BaseEigenState struct {
}

// Map of block number to function that will transition the state to the next block
type StateTransitions[T interface{}] map[uint64]func(log *storage.TransactionLog) (*T, error)
