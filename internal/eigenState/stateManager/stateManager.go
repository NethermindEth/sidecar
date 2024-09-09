package stateManager

import (
	"fmt"
	"github.com/Layr-Labs/go-sidecar/internal/eigenState/types"
	"github.com/Layr-Labs/go-sidecar/internal/storage"
	"github.com/Layr-Labs/go-sidecar/internal/utils"
	"github.com/wealdtech/go-merkletree/v2"
	"github.com/wealdtech/go-merkletree/v2/keccak256"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"slices"
	"time"
)

type StateRoot struct {
	EthBlockNumber uint64
	EthBlockHash   string
	StateRoot      string
	CreatedAt      time.Time
}

type EigenStateManager struct {
	StateModels map[int]types.IEigenStateModel
	logger      *zap.Logger
	Db          *gorm.DB
}

func NewEigenStateManager(logger *zap.Logger, grm *gorm.DB) *EigenStateManager {
	return &EigenStateManager{
		StateModels: make(map[int]types.IEigenStateModel),
		logger:      logger,
		Db:          grm,
	}
}

// Allows a model to register itself with the state manager
func (e *EigenStateManager) RegisterState(model types.IEigenStateModel, index int) {
	if m, ok := e.StateModels[index]; ok {
		e.logger.Sugar().Fatalf("Registering model model at index %d which already exists and belongs to %s", index, m.GetModelName())
	}
	e.StateModels[index] = model
}

// Given a log, allow each state model to determine if/how to process it
func (e *EigenStateManager) HandleLogStateChange(log *storage.TransactionLog) error {
	e.logger.Sugar().Debugw("Handling log state change", zap.String("transactionHash", log.TransactionHash), zap.Uint64("logIndex", log.LogIndex))
	for _, index := range e.GetSortedModelIndexes() {
		state := e.StateModels[index]
		if state.IsInterestingLog(log) {
			e.logger.Sugar().Debugw("Handling log for model",
				zap.String("model", state.GetModelName()),
				zap.String("transactionHash", log.TransactionHash),
				zap.Uint64("logIndex", log.LogIndex),
				zap.String("eventName", log.EventName),
			)
			_, err := state.HandleStateChange(log)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (e *EigenStateManager) InitProcessingForBlock(blockNumber uint64) error {
	for _, index := range e.GetSortedModelIndexes() {
		state := e.StateModels[index]
		err := state.InitBlockProcessing(blockNumber)
		if err != nil {
			return err
		}
	}
	return nil

}

// With all transactions/logs processed for a block, commit the final state to the table
func (e *EigenStateManager) CommitFinalState(blockNumber uint64) error {
	for _, index := range e.GetSortedModelIndexes() {
		state := e.StateModels[index]
		err := state.CommitFinalState(blockNumber)
		if err != nil {
			return err
		}
	}
	return nil
}

func (e *EigenStateManager) CleanupBlock(blockNumber uint64) error {
	for _, index := range e.GetSortedModelIndexes() {
		state := e.StateModels[index]
		err := state.ClearAccumulatedState(blockNumber)
		if err != nil {
			return err
		}
	}
	return nil
}

func (e *EigenStateManager) GenerateStateRoot(blockNumber uint64, blockHash string) (types.StateRoot, error) {
	sortedIndexes := e.GetSortedModelIndexes()
	roots := [][]byte{
		[]byte(fmt.Sprintf("%d", blockNumber)),
		[]byte(blockHash),
	}

	for _, state := range sortedIndexes {
		state := e.StateModels[state]
		leaf, err := e.encodeModelLeaf(state, blockNumber)
		if err != nil {
			return "", err
		}
		roots = append(roots, leaf)
	}

	tree, err := merkletree.NewTree(
		merkletree.WithData(roots),
		merkletree.WithHashType(keccak256.New()),
	)
	if err != nil {
		return "", err
	}

	return types.StateRoot(utils.ConvertBytesToString(tree.Root())), nil
}

func (e *EigenStateManager) WriteStateRoot(
	blockNumber uint64,
	blockHash string,
	stateroot types.StateRoot,
) (*StateRoot, error) {
	root := &StateRoot{
		EthBlockNumber: blockNumber,
		EthBlockHash:   blockHash,
		StateRoot:      string(stateroot),
	}

	result := e.Db.Model(&StateRoot{}).Clauses(clause.Returning{}).Create(&root)
	if result.Error != nil {
		return nil, result.Error
	}
	return root, nil
}

func (e *EigenStateManager) GetStateRootForBlock(blockNumber uint64) (*StateRoot, error) {
	root := &StateRoot{}
	result := e.Db.Model(&StateRoot{}).Where("eth_block_number = ?", blockNumber).First(&root)
	if result.Error != nil {
		return nil, result.Error
	}
	return root, nil
}

func (e *EigenStateManager) encodeModelLeaf(model types.IEigenStateModel, blockNumber uint64) ([]byte, error) {
	root, err := model.GenerateStateRoot(blockNumber)
	if err != nil {
		return nil, err
	}
	return append([]byte(model.GetModelName()), []byte(root)[:]...), nil
}

func (e *EigenStateManager) GetSortedModelIndexes() []int {
	indexes := make([]int, 0, len(e.StateModels))
	for i := range e.StateModels {
		indexes = append(indexes, i)
	}
	slices.Sort(indexes)
	return indexes
}
