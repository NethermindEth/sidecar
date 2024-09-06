package eigenState

import (
	"github.com/Layr-Labs/sidecar/internal/storage"
	"go.uber.org/zap"
)

type EigenStateManager struct {
	StateModels map[int]IEigenStateModel
	logger      *zap.Logger
}

func NewEigenStateManager(logger *zap.Logger) *EigenStateManager {
	return &EigenStateManager{
		StateModels: make(map[int]IEigenStateModel),
		logger:      logger,
	}
}

// Allows a model to register itself with the state manager
func (e *EigenStateManager) RegisterState(model IEigenStateModel, index int) {
	if m, ok := e.StateModels[index]; ok {
		e.logger.Sugar().Fatalf("Registering model model at index %d which already exists and belongs to %s", index, m.GetModelName())
	}
	e.StateModels[index] = model
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
