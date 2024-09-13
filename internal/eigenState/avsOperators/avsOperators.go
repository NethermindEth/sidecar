package avsOperators

import (
	"database/sql"
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/Layr-Labs/go-sidecar/internal/config"
	"github.com/Layr-Labs/go-sidecar/internal/eigenState/base"
	"github.com/Layr-Labs/go-sidecar/internal/eigenState/stateManager"
	"github.com/Layr-Labs/go-sidecar/internal/eigenState/types"
	"github.com/Layr-Labs/go-sidecar/internal/storage"
	"github.com/Layr-Labs/go-sidecar/internal/utils"
	"github.com/wealdtech/go-merkletree/v2"
	"github.com/wealdtech/go-merkletree/v2/keccak256"
	orderedmap "github.com/wk8/go-ordered-map/v2"
	"go.uber.org/zap"
	"golang.org/x/xerrors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Schema for registered_avs_operators block state table.
type RegisteredAvsOperators struct {
	Operator    string
	Avs         string
	BlockNumber uint64
	CreatedAt   time.Time
}

// AccumulatedStateChange represents the accumulated state change for a given block.
type AccumulatedStateChange struct {
	Avs         string
	Operator    string
	Registered  bool
	BlockNumber uint64
}

// RegisteredAvsOperatorDiff represents the diff between the registered_avs_operators table and the accumulated state.
type RegisteredAvsOperatorDiff struct {
	Avs         string
	Operator    string
	BlockNumber uint64
	Registered  bool
}

func NewSlotID(avs string, operator string) types.SlotID {
	return types.SlotID(fmt.Sprintf("%s_%s", avs, operator))
}

// EigenState model for AVS operators that implements IEigenStateModel.
type AvsOperatorsModel struct {
	base.BaseEigenState
	StateTransitions types.StateTransitions[AccumulatedStateChange]
	Db               *gorm.DB
	Network          config.Network
	Environment      config.Environment
	logger           *zap.Logger
	globalConfig     *config.Config

	// Accumulates state changes for SlotIds, grouped by block number
	stateAccumulator map[uint64]map[types.SlotID]*AccumulatedStateChange
}

// Create new instance of AvsOperatorsModel state model.
func NewAvsOperators(
	esm *stateManager.EigenStateManager,
	grm *gorm.DB,
	Network config.Network,
	Environment config.Environment,
	logger *zap.Logger,
	globalConfig *config.Config,
) (*AvsOperatorsModel, error) {
	s := &AvsOperatorsModel{
		BaseEigenState: base.BaseEigenState{
			Logger: logger,
		},
		Db:           grm,
		Network:      Network,
		Environment:  Environment,
		logger:       logger,
		globalConfig: globalConfig,

		stateAccumulator: make(map[uint64]map[types.SlotID]*AccumulatedStateChange),
	}
	esm.RegisterState(s, 0)
	return s, nil
}

func (a *AvsOperatorsModel) GetModelName() string {
	return "AvsOperatorsModel"
}

// Get the state transitions for the AvsOperatorsModel state model
//
// Each state transition is function indexed by a block number.
// BlockNumber 0 is the catchall state
//
// Returns the map and a reverse sorted list of block numbers that can be traversed when
// processing a log to determine which state change to apply.
func (a *AvsOperatorsModel) GetStateTransitions() (types.StateTransitions[AccumulatedStateChange], []uint64) {
	stateChanges := make(types.StateTransitions[AccumulatedStateChange])

	// TODO(seanmcgary): make this not a closure so this function doesnt get big an messy...
	stateChanges[0] = func(log *storage.TransactionLog) (*AccumulatedStateChange, error) {
		arguments, err := a.ParseLogArguments(log)
		if err != nil {
			return nil, err
		}

		outputData, err := a.ParseLogOutput(log)
		if err != nil {
			return nil, err
		}

		// Sanity check to make sure we've got an initialized accumulator map for the block
		if _, ok := a.stateAccumulator[log.BlockNumber]; !ok {
			return nil, xerrors.Errorf("No state accumulator found for block %d", log.BlockNumber)
		}

		avs := strings.ToLower(arguments[0].Value.(string))
		operator := strings.ToLower(arguments[1].Value.(string))

		registered := false
		if val, ok := outputData["status"]; ok {
			registered = uint64(val.(float64)) == 1
		}

		slotID := NewSlotID(avs, operator)
		record, ok := a.stateAccumulator[log.BlockNumber][slotID]
		if !ok {
			record = &AccumulatedStateChange{
				Avs:         avs,
				Operator:    operator,
				BlockNumber: log.BlockNumber,
			}
			a.stateAccumulator[log.BlockNumber][slotID] = record
		}
		if registered == false && ok {
			// In this situation, we've encountered a register and unregister in the same block
			// which functionally results in no state change at all so we want to remove the record
			// from the accumulated state.
			delete(a.stateAccumulator[log.BlockNumber], slotID)
			return nil, nil
		}
		record.Registered = registered

		return record, nil
	}

	// Create an ordered list of block numbers
	blockNumbers := make([]uint64, 0)
	for blockNumber := range stateChanges {
		blockNumbers = append(blockNumbers, blockNumber)
	}
	sort.Slice(blockNumbers, func(i, j int) bool {
		return blockNumbers[i] < blockNumbers[j]
	})
	slices.Reverse(blockNumbers)

	return stateChanges, blockNumbers
}

// Returns a map of contract addresses to event names that are interesting to the state model.
func (a *AvsOperatorsModel) getContractAddressesForEnvironment() map[string][]string {
	contracts := a.globalConfig.GetContractsMapForEnvAndNetwork()
	return map[string][]string{
		contracts.AvsDirectory: {
			"OperatorAVSRegistrationStatusUpdated",
		},
	}
}

// Given a log, determine if it is interesting to the state model.
func (a *AvsOperatorsModel) IsInterestingLog(log *storage.TransactionLog) bool {
	addresses := a.getContractAddressesForEnvironment()
	return a.BaseEigenState.IsInterestingLog(addresses, log)
}

func (a *AvsOperatorsModel) InitBlockProcessing(blockNumber uint64) error {
	a.stateAccumulator[blockNumber] = make(map[types.SlotID]*AccumulatedStateChange)
	return nil
}

// Handle the state change for the given log
//
// Takes a log and iterates over the state transitions to determine which state change to apply based on block number.
func (a *AvsOperatorsModel) HandleStateChange(log *storage.TransactionLog) (interface{}, error) {
	stateChanges, sortedBlockNumbers := a.GetStateTransitions()

	for _, blockNumber := range sortedBlockNumbers {
		if log.BlockNumber >= blockNumber {
			a.logger.Sugar().Debugw("Handling state change", zap.Uint64("blockNumber", blockNumber))

			change, err := stateChanges[blockNumber](log)
			if err != nil {
				return nil, err
			}

			if change == nil {
				return nil, xerrors.Errorf("No state change found for block %d", blockNumber)
			}
			return change, nil
		}
	}
	return nil, nil
}

func (a *AvsOperatorsModel) clonePreviousBlocksToNewBlock(blockNumber uint64) error {
	query := `
		insert into registered_avs_operators (avs, operator, block_number)
			select
				avs,
				operator,
				@currentBlock as block_number
			from registered_avs_operators
			where block_number = @previousBlock
	`
	res := a.Db.Exec(query,
		sql.Named("currentBlock", blockNumber),
		sql.Named("previousBlock", blockNumber-1),
	)

	if res.Error != nil {
		a.logger.Sugar().Errorw("Failed to clone previous block state to new block", zap.Error(res.Error))
		return res.Error
	}
	return nil
}

// prepareState prepares the state for the current block by comparing the accumulated state changes.
// It separates out the changes into inserts and deletes.
func (a *AvsOperatorsModel) prepareState(blockNumber uint64) ([]RegisteredAvsOperators, []RegisteredAvsOperators, error) {
	accumulatedState, ok := a.stateAccumulator[blockNumber]
	if !ok {
		err := xerrors.Errorf("No accumulated state found for block %d", blockNumber)
		a.logger.Sugar().Errorw(err.Error(), zap.Error(err), zap.Uint64("blockNumber", blockNumber))
		return nil, nil, err
	}

	inserts := make([]RegisteredAvsOperators, 0)
	deletes := make([]RegisteredAvsOperators, 0)
	for _, stateChange := range accumulatedState {
		record := RegisteredAvsOperators{
			Avs:         stateChange.Avs,
			Operator:    stateChange.Operator,
			BlockNumber: blockNumber,
		}
		if stateChange.Registered {
			inserts = append(inserts, record)
		} else {
			deletes = append(deletes, record)
		}
	}
	return inserts, deletes, nil
}

// CommitFinalState commits the final state for the given block number.
func (a *AvsOperatorsModel) CommitFinalState(blockNumber uint64) error {
	err := a.clonePreviousBlocksToNewBlock(blockNumber)
	if err != nil {
		return err
	}

	recordsToInsert, recordsToDelete, err := a.prepareState(blockNumber)
	if err != nil {
		return err
	}

	for _, record := range recordsToDelete {
		res := a.Db.Delete(&RegisteredAvsOperators{}, "avs = ? and operator = ? and block_number = ?", record.Avs, record.Operator, record.BlockNumber)
		if res.Error != nil {
			a.logger.Sugar().Errorw("Failed to delete record",
				zap.Error(res.Error),
				zap.String("avs", record.Avs),
				zap.String("operator", record.Operator),
				zap.Uint64("blockNumber", blockNumber),
			)
			return res.Error
		}
	}
	if len(recordsToInsert) > 0 {
		res := a.Db.Model(&RegisteredAvsOperators{}).Clauses(clause.Returning{}).Create(&recordsToInsert)
		if res.Error != nil {
			a.logger.Sugar().Errorw("Failed to insert records", zap.Error(res.Error))
			return res.Error
		}
	}
	return nil
}

func (a *AvsOperatorsModel) ClearAccumulatedState(blockNumber uint64) error {
	delete(a.stateAccumulator, blockNumber)
	return nil
}

// GenerateStateRoot generates the state root for the given block number using the results of the state changes.
func (a *AvsOperatorsModel) GenerateStateRoot(blockNumber uint64) (types.StateRoot, error) {
	inserts, deletes, err := a.prepareState(blockNumber)
	if err != nil {
		return "", err
	}

	combinedResults := make([]RegisteredAvsOperatorDiff, 0)
	for _, record := range inserts {
		combinedResults = append(combinedResults, RegisteredAvsOperatorDiff{
			Avs:         record.Avs,
			Operator:    record.Operator,
			BlockNumber: record.BlockNumber,
			Registered:  true,
		})
	}
	for _, record := range deletes {
		combinedResults = append(combinedResults, RegisteredAvsOperatorDiff{
			Avs:         record.Avs,
			Operator:    record.Operator,
			BlockNumber: record.BlockNumber,
			Registered:  false,
		})
	}

	fullTree, err := a.merkelizeState(blockNumber, combinedResults)
	if err != nil {
		return "", err
	}
	return types.StateRoot(utils.ConvertBytesToString(fullTree.Root())), nil
}

func (a *AvsOperatorsModel) merkelizeState(blockNumber uint64, avsOperators []RegisteredAvsOperatorDiff) (*merkletree.MerkleTree, error) {
	// Avs -> operator:registered
	om := orderedmap.New[string, *orderedmap.OrderedMap[string, bool]]()

	for _, result := range avsOperators {
		existingAvs, found := om.Get(result.Avs)
		if !found {
			existingAvs = orderedmap.New[string, bool]()
			om.Set(result.Avs, existingAvs)

			prev := om.GetPair(result.Avs).Prev()
			if prev != nil && strings.Compare(prev.Key, result.Avs) >= 0 {
				om.Delete(result.Avs)
				return nil, fmt.Errorf("avs not in order")
			}
		}
		existingAvs.Set(result.Operator, result.Registered)

		prev := existingAvs.GetPair(result.Operator).Prev()
		if prev != nil && strings.Compare(prev.Key, result.Operator) >= 0 {
			existingAvs.Delete(result.Operator)
			return nil, fmt.Errorf("operator not in order")
		}
	}

	avsLeaves := a.InitializeMerkleTreeBaseStateWithBlock(blockNumber)

	for avs := om.Oldest(); avs != nil; avs = avs.Next() {
		operatorLeafs := make([][]byte, 0)
		for operator := avs.Value.Oldest(); operator != nil; operator = operator.Next() {
			operatorAddr := operator.Key
			registered := operator.Value
			operatorLeafs = append(operatorLeafs, encodeOperatorLeaf(operatorAddr, registered))
		}

		avsTree, err := merkletree.NewTree(
			merkletree.WithData(operatorLeafs),
			merkletree.WithHashType(keccak256.New()),
		)
		if err != nil {
			return nil, err
		}

		avsLeaves = append(avsLeaves, encodeAvsLeaf(avs.Key, avsTree.Root()))
	}

	return merkletree.NewTree(
		merkletree.WithData(avsLeaves),
		merkletree.WithHashType(keccak256.New()),
	)
}

func (a *AvsOperatorsModel) DeleteState(startBlockNumber uint64, endBlockNumber uint64) error {
	return a.BaseEigenState.DeleteState("registered_avs_operators", startBlockNumber, endBlockNumber, a.Db)
}

func encodeOperatorLeaf(operator string, registered bool) []byte {
	return []byte(fmt.Sprintf("%s:%t", operator, registered))
}

func encodeAvsLeaf(avs string, avsOperatorRoot []byte) []byte {
	return append([]byte(avs), avsOperatorRoot...)
}
