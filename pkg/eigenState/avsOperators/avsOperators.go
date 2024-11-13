package avsOperators

import (
	"errors"
	"fmt"
	"github.com/Layr-Labs/sidecar/internal/config"
	"github.com/Layr-Labs/sidecar/pkg/eigenState/base"
	"github.com/Layr-Labs/sidecar/pkg/eigenState/stateManager"
	"github.com/Layr-Labs/sidecar/pkg/eigenState/types"
	"github.com/Layr-Labs/sidecar/pkg/storage"
	"github.com/Layr-Labs/sidecar/pkg/utils"
	"go.uber.org/zap"
	"golang.org/x/xerrors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"slices"
	"sort"
	"strings"
)

type AvsOperatorStateChange struct {
	Avs         string
	Operator    string
	Registered  bool
	LogIndex    uint64
	BlockNumber uint64
}

func NewSlotID(avs string, operator string, logIndex uint64) types.SlotID {
	return types.SlotID(fmt.Sprintf("%s_%s_%d", avs, operator, logIndex))
}

// EigenState model for AVS operators that implements IEigenStateModel.
type AvsOperatorsModel struct {
	base.BaseEigenState
	StateTransitions types.StateTransitions[AvsOperatorStateChange]
	DB               *gorm.DB
	logger           *zap.Logger
	globalConfig     *config.Config

	// Keep track of each distinct change, rather than accumulated change, to add to the delta table
	stateAccumulator map[uint64][]*AvsOperatorStateChange
}

// NewAvsOperatorsModel creates a new AvsOperatorsModel.
func NewAvsOperatorsModel(
	esm *stateManager.EigenStateManager,
	grm *gorm.DB,
	logger *zap.Logger,
	globalConfig *config.Config,
) (*AvsOperatorsModel, error) {
	s := &AvsOperatorsModel{
		BaseEigenState: base.BaseEigenState{
			Logger: logger,
		},
		DB:           grm,
		logger:       logger,
		globalConfig: globalConfig,

		stateAccumulator: make(map[uint64][]*AvsOperatorStateChange),
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
func (a *AvsOperatorsModel) GetStateTransitions() (types.StateTransitions[*AvsOperatorStateChange], []uint64) {
	stateChanges := make(types.StateTransitions[*AvsOperatorStateChange])

	stateChanges[0] = func(log *storage.TransactionLog) (*AvsOperatorStateChange, error) {
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

		operator := strings.ToLower(arguments[0].Value.(string))
		avs := strings.ToLower(arguments[1].Value.(string))

		registered := false
		if val, ok := outputData["status"]; ok {
			registered = uint64(val.(float64)) == 1
		}

		// Store the change in the delta accumulator
		delta := &AvsOperatorStateChange{
			Avs:         avs,
			Operator:    operator,
			Registered:  registered,
			LogIndex:    log.LogIndex,
			BlockNumber: log.BlockNumber,
		}
		a.stateAccumulator[log.BlockNumber] = append(a.stateAccumulator[log.BlockNumber], delta)

		return delta, nil
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
	contracts := a.globalConfig.GetContractsMapForChain()
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

func (a *AvsOperatorsModel) SetupStateForBlock(blockNumber uint64) error {
	a.stateAccumulator[blockNumber] = make([]*AvsOperatorStateChange, 0)
	return nil
}

func (a *AvsOperatorsModel) CleanupProcessedStateForBlock(blockNumber uint64) error {
	delete(a.stateAccumulator, blockNumber)
	return nil
}

// Handle the state change for the given log
//
// Takes a log and iterates over the state transitions to determine which state change to apply based on block number.
func (a *AvsOperatorsModel) HandleStateChange(log *storage.TransactionLog) (interface{}, error) {
	stateChanges, sortedBlockNumbers := a.GetStateTransitions()

	for _, blockNumber := range sortedBlockNumbers {
		if log.BlockNumber >= blockNumber {
			a.logger.Sugar().Debugw("Handling state change", zap.Uint64("blockNumber", log.BlockNumber))

			change, err := stateChanges[blockNumber](log)
			if err != nil {
				return nil, err
			}

			if change == nil {
				a.logger.Sugar().Debugw("No state change found", zap.Uint64("blockNumber", blockNumber))
				return nil, nil
			}
			return change, nil
		}
	}
	return nil, nil
}

// prepareState prepares the state for the current block by comparing the accumulated state changes.
// It separates out the changes into inserts and deletes.
func (a *AvsOperatorsModel) prepareState(blockNumber uint64) ([]*AvsOperatorStateChange, error) {
	accumulatedState, ok := a.stateAccumulator[blockNumber]
	if !ok {
		err := xerrors.Errorf("No accumulated state found for block %d", blockNumber)
		a.logger.Sugar().Errorw(err.Error(), zap.Error(err), zap.Uint64("blockNumber", blockNumber))
		return nil, err
	}

	return accumulatedState, nil
}

func (a *AvsOperatorsModel) writeDeltaRecords(blockNumber uint64) error {
	records, ok := a.stateAccumulator[blockNumber]
	if !ok {
		msg := "delta accumulator was not initialized"
		a.logger.Sugar().Errorw(msg, zap.Uint64("blockNumber", blockNumber))
		return errors.New(msg)
	}

	if len(records) > 0 {
		res := a.DB.Model(&AvsOperatorStateChange{}).Clauses(clause.Returning{}).Create(&records)
		if res.Error != nil {
			a.logger.Sugar().Errorw("Failed to insert delta records", zap.Error(res.Error))
			return res.Error
		}
	}
	return nil
}

// CommitFinalState commits the final state for the given block number.
func (a *AvsOperatorsModel) CommitFinalState(blockNumber uint64) error {
	if err := a.writeDeltaRecords(blockNumber); err != nil {
		return err
	}

	return nil
}

// GenerateStateRoot generates the state root for the given block number using the results of the state changes.
func (a *AvsOperatorsModel) GenerateStateRoot(blockNumber uint64) (types.StateRoot, error) {
	deltas, err := a.prepareState(blockNumber)
	if err != nil {
		return "", err
	}

	inputs := a.sortValuesForMerkleTree(deltas)

	if len(inputs) == 0 {
		return "", nil
	}

	fullTree, err := a.MerkleizeState(blockNumber, inputs)
	if err != nil {
		a.logger.Sugar().Errorw("Failed to create merkle tree",
			zap.Error(err),
			zap.Uint64("blockNumber", blockNumber),
			zap.Any("inputs", inputs),
		)
		return "", err
	}
	return types.StateRoot(utils.ConvertBytesToString(fullTree.Root())), nil
}

func (a *AvsOperatorsModel) sortValuesForMerkleTree(diffs []*AvsOperatorStateChange) []*base.MerkleTreeInput {
	inputs := make([]*base.MerkleTreeInput, 0)
	for _, diff := range diffs {
		inputs = append(inputs, &base.MerkleTreeInput{
			SlotID: NewSlotID(diff.Avs, diff.Operator, diff.LogIndex),
			Value:  []byte(fmt.Sprintf("%t", diff.Registered)),
		})
	}
	slices.SortFunc(inputs, func(i, j *base.MerkleTreeInput) int {
		return strings.Compare(string(i.SlotID), string(j.SlotID))
	})
	return inputs
}

func (a *AvsOperatorsModel) DeleteState(startBlockNumber uint64, endBlockNumber uint64) error {
	return a.BaseEigenState.DeleteState("avs_operator_state_changes", startBlockNumber, endBlockNumber, a.DB)
}
