package stakerDelegations

import (
	"database/sql"
	"fmt"
	"github.com/Layr-Labs/sidecar/internal/config"
	"github.com/Layr-Labs/sidecar/internal/eigenState/base"
	"github.com/Layr-Labs/sidecar/internal/eigenState/stateManager"
	"github.com/Layr-Labs/sidecar/internal/eigenState/types"
	"github.com/Layr-Labs/sidecar/internal/storage"
	"github.com/Layr-Labs/sidecar/internal/utils"
	"github.com/wealdtech/go-merkletree/v2"
	"github.com/wealdtech/go-merkletree/v2/keccak256"
	orderedmap "github.com/wk8/go-ordered-map/v2"
	"go.uber.org/zap"
	"golang.org/x/xerrors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"slices"
	"sort"
	"strings"
	"time"
)

type DelegatedStakers struct {
	Staker      string
	Operator    string
	BlockNumber uint64
	CreatedAt   time.Time
}

type StakerDelegationChange struct {
	Staker           string
	Operator         string
	Delegated        bool
	TransactionHash  string
	TransactionIndex uint64
	LogIndex         uint64
	BlockNumber      uint64
	CreatedAt        time.Time
}

type AccumulatedStateChange struct {
	Staker      string
	Operator    string
	BlockNumber uint64
	Delegated   bool
}

type SlotId string

func NewSlotId(staker string, operator string) SlotId {
	return SlotId(fmt.Sprintf("%s_%s", staker, operator))
}

type StakerDelegationsModel struct {
	base.BaseEigenState
	StateTransitions types.StateTransitions[StakerDelegationChange]
	Db               *gorm.DB
	Network          config.Network
	Environment      config.Environment
	logger           *zap.Logger
	globalConfig     *config.Config

	// Accumulates state changes for SlotIds, grouped by block number
	stateAccumulator map[uint64]map[SlotId]*AccumulatedStateChange
}

type DelegatedStakersDiff struct {
	Staker      string
	Operator    string
	Delegated   bool
	BlockNumber uint64
}

func NewStakerDelegationsModel(
	esm *stateManager.EigenStateManager,
	grm *gorm.DB,
	Network config.Network,
	Environment config.Environment,
	logger *zap.Logger,
	globalConfig *config.Config,
) (*StakerDelegationsModel, error) {
	model := &StakerDelegationsModel{
		BaseEigenState: base.BaseEigenState{
			Logger: logger,
		},
		Db:               grm,
		Network:          Network,
		Environment:      Environment,
		logger:           logger,
		globalConfig:     globalConfig,
		stateAccumulator: make(map[uint64]map[SlotId]*AccumulatedStateChange),
	}

	esm.RegisterState(model, 2)
	return model, nil
}

func (s *StakerDelegationsModel) GetModelName() string {
	return "StakerDelegationsModel"
}

func (s *StakerDelegationsModel) GetStateTransitions() (types.StateTransitions[AccumulatedStateChange], []uint64) {
	stateChanges := make(types.StateTransitions[AccumulatedStateChange])

	stateChanges[0] = func(log *storage.TransactionLog) (*AccumulatedStateChange, error) {
		arguments, err := s.ParseLogArguments(log)
		if err != nil {
			return nil, err
		}

		// Sanity check to make sure we've got an initialized accumulator map for the block
		if _, ok := s.stateAccumulator[log.BlockNumber]; !ok {
			return nil, xerrors.Errorf("No state accumulator found for block %d", log.BlockNumber)
		}

		staker := arguments[0].Value.(string)
		operator := arguments[1].Value.(string)

		slotId := NewSlotId(staker, operator)
		record, ok := s.stateAccumulator[log.BlockNumber][slotId]
		if !ok {
			// if the record doesn't exist, create a new one
			record = &AccumulatedStateChange{
				Staker:      staker,
				Operator:    operator,
				BlockNumber: log.BlockNumber,
			}
			s.stateAccumulator[log.BlockNumber][slotId] = record
		}
		if log.EventName == "StakerUndelegated" {
			if ok {
				// In this situation, we've encountered a delegate and undelegate in the same block
				// which functionally results in no state change at all so we want to remove the record
				// from the accumulated state.
				delete(s.stateAccumulator[log.BlockNumber], slotId)
				return nil, nil
			}
			record.Delegated = false
		} else if log.EventName == "StakerDelegated" {
			record.Delegated = true
		}

		return record, nil
	}

	// Create an ordered list of block numbers
	blockNumbers := make([]uint64, 0)
	for blockNumber, _ := range stateChanges {
		blockNumbers = append(blockNumbers, blockNumber)
	}
	sort.Slice(blockNumbers, func(i, j int) bool {
		return blockNumbers[i] < blockNumbers[j]
	})
	slices.Reverse(blockNumbers)

	return stateChanges, blockNumbers
}

func (s *StakerDelegationsModel) getContractAddressesForEnvironment() map[string][]string {
	contracts := s.globalConfig.GetContractsMapForEnvAndNetwork()
	return map[string][]string{
		contracts.DelegationManager: []string{
			"StakerUndelegated",
			"StakerDelegated",
		},
	}
}

func (s *StakerDelegationsModel) IsInterestingLog(log *storage.TransactionLog) bool {
	addresses := s.getContractAddressesForEnvironment()
	return s.BaseEigenState.IsInterestingLog(addresses, log)
}

// StartBlockProcessing Initialize state accumulator for the block
func (s *StakerDelegationsModel) StartBlockProcessing(blockNumber uint64) error {
	s.stateAccumulator[blockNumber] = make(map[SlotId]*AccumulatedStateChange)
	return nil
}

func (s *StakerDelegationsModel) HandleStateChange(log *storage.TransactionLog) (interface{}, error) {
	stateChanges, sortedBlockNumbers := s.GetStateTransitions()

	for _, blockNumber := range sortedBlockNumbers {
		if log.BlockNumber >= blockNumber {
			s.logger.Sugar().Debugw("Handling state change", zap.Uint64("blockNumber", blockNumber))

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

func (s *StakerDelegationsModel) clonePreviousBlocksToNewBlock(blockNumber uint64) error {
	query := `
		insert into delegated_stakers (staker, operator, block_number)
			select
				staker,
				operator,
				@currentBlock as block_number
			from delegated_stakers
			where block_number = @previousBlock
	`
	res := s.Db.Exec(query,
		sql.Named("currentBlock", blockNumber),
		sql.Named("previousBlock", blockNumber-1),
	)

	if res.Error != nil {
		s.logger.Sugar().Errorw("Failed to clone previous block state to new block", zap.Error(res.Error))
		return res.Error
	}
	return nil
}

// prepareState prepares the state for the current block by comparing the accumulated state changes.
// It separates out the changes into inserts and deletes
func (s *StakerDelegationsModel) prepareState(blockNumber uint64) ([]DelegatedStakers, []DelegatedStakers, error) {
	accumulatedState, ok := s.stateAccumulator[blockNumber]
	if !ok {
		err := xerrors.Errorf("No accumulated state found for block %d", blockNumber)
		s.logger.Sugar().Errorw(err.Error(), zap.Error(err), zap.Uint64("blockNumber", blockNumber))
		return nil, nil, err
	}

	inserts := make([]DelegatedStakers, 0)
	deletes := make([]DelegatedStakers, 0)
	for _, stateChange := range accumulatedState {
		record := DelegatedStakers{
			Staker:      stateChange.Staker,
			Operator:    stateChange.Operator,
			BlockNumber: blockNumber,
		}
		if stateChange.Delegated {
			inserts = append(inserts, record)
		} else {
			deletes = append(deletes, record)
		}
	}
	return inserts, deletes, nil
}

func (s *StakerDelegationsModel) CommitFinalState(blockNumber uint64) error {
	// Clone the previous block state to give us a reference point.
	//
	// By doing this, existing staker delegations will be carried over to the new block.
	// We'll then remove any stakers that were undelegated and add any new stakers that were delegated.
	err := s.clonePreviousBlocksToNewBlock(blockNumber)
	if err != nil {
		return err
	}

	recordsToInsert, recordsToDelete, err := s.prepareState(blockNumber)

	// TODO(seanmcgary): should probably wrap the operations of this function in a db transaction
	for _, record := range recordsToDelete {
		res := s.Db.Delete(&DelegatedStakers{}, "staker = ? and operator = ? and block_number = ?", record.Staker, record.Operator, blockNumber)
		if res.Error != nil {
			s.logger.Sugar().Errorw("Failed to delete staker delegation",
				zap.Error(res.Error),
				zap.String("staker", record.Staker),
				zap.String("operator", record.Operator),
				zap.Uint64("blockNumber", blockNumber),
			)
			return res.Error
		}
	}
	if len(recordsToInsert) > 0 {
		res := s.Db.Model(&DelegatedStakers{}).Clauses(clause.Returning{}).Create(&recordsToInsert)
		if res.Error != nil {
			s.logger.Sugar().Errorw("Failed to insert staker delegations", zap.Error(res.Error))
			return res.Error
		}
	}
	return nil
}

// ClearAccumulatedState clears the accumulated state for the given block number to free up memory
func (s *StakerDelegationsModel) ClearAccumulatedState(blockNumber uint64) error {
	delete(s.stateAccumulator, blockNumber)
	return nil
}

// GenerateStateRoot generates the state root for the given block number by storing
// the state changes in a merkle tree.
func (s *StakerDelegationsModel) GenerateStateRoot(blockNumber uint64) (types.StateRoot, error) {
	inserts, deletes, err := s.prepareState(blockNumber)
	if err != nil {
		return "", err
	}

	// Take all of the inserts and deletes and combine them into a single list
	combinedResults := make([]DelegatedStakersDiff, 0)
	for _, record := range inserts {
		combinedResults = append(combinedResults, DelegatedStakersDiff{
			Staker:      record.Staker,
			Operator:    record.Operator,
			Delegated:   true,
			BlockNumber: blockNumber,
		})
	}
	for _, record := range deletes {
		combinedResults = append(combinedResults, DelegatedStakersDiff{
			Staker:      record.Staker,
			Operator:    record.Operator,
			Delegated:   false,
			BlockNumber: blockNumber,
		})
	}

	fullTree, err := s.merkelizeState(blockNumber, combinedResults)
	if err != nil {
		return "", err
	}
	return types.StateRoot(utils.ConvertBytesToString(fullTree.Root())), nil
}

// merkelizeState generates a merkle tree for the given block number and delegated stakers.
// Changes are stored in the following format:
// Operator -> staker:delegated
func (s *StakerDelegationsModel) merkelizeState(blockNumber uint64, delegatedStakers []DelegatedStakersDiff) (*merkletree.MerkleTree, error) {
	om := orderedmap.New[string, *orderedmap.OrderedMap[string, bool]]()

	for _, result := range delegatedStakers {
		existingOperator, found := om.Get(result.Operator)
		if !found {
			existingOperator = orderedmap.New[string, bool]()
			om.Set(result.Operator, existingOperator)

			prev := om.GetPair(result.Operator).Prev()
			if prev != nil && strings.Compare(prev.Key, result.Operator) >= 0 {
				om.Delete(result.Operator)
				return nil, fmt.Errorf("operators not in order")
			}
		}
		existingOperator.Set(result.Staker, result.Delegated)

		prev := existingOperator.GetPair(result.Staker).Prev()
		if prev != nil && strings.Compare(prev.Key, result.Staker) >= 0 {
			existingOperator.Delete(result.Staker)
			return nil, fmt.Errorf("stakers not in order")
		}
	}

	operatorLeaves := s.InitializeMerkleTreeBaseStateWithBlock(blockNumber)

	for op := om.Oldest(); op != nil; op = op.Next() {

		stakerLeafs := make([][]byte, 0)
		for staker := op.Value.Oldest(); staker != nil; staker = staker.Next() {
			operatorAddr := staker.Key
			delegated := staker.Value
			stakerLeafs = append(stakerLeafs, encodeStakerLeaf(operatorAddr, delegated))
		}

		avsTree, err := merkletree.NewTree(
			merkletree.WithData(stakerLeafs),
			merkletree.WithHashType(keccak256.New()),
		)
		if err != nil {
			return nil, err
		}

		operatorLeaves = append(operatorLeaves, encodeOperatorLeaf(op.Key, avsTree.Root()))
	}

	return merkletree.NewTree(
		merkletree.WithData(operatorLeaves),
		merkletree.WithHashType(keccak256.New()),
	)
}

func encodeStakerLeaf(staker string, delegated bool) []byte {
	return []byte(fmt.Sprintf("%s:%t", staker, delegated))
}

func encodeOperatorLeaf(operator string, operatorStakersRoot []byte) []byte {
	return append([]byte(operator), operatorStakersRoot[:]...)
}
