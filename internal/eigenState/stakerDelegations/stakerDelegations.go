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

type StakerDelegationsModel struct {
	base.BaseEigenState
	StateTransitions types.StateTransitions[StakerDelegationChange]
	Db               *gorm.DB
	Network          config.Network
	Environment      config.Environment
	logger           *zap.Logger
	globalConfig     *config.Config
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
		Db:           grm,
		Network:      Network,
		Environment:  Environment,
		logger:       logger,
		globalConfig: globalConfig,
	}

	esm.RegisterState(model, 2)
	return model, nil
}

func (s *StakerDelegationsModel) GetModelName() string {
	return "StakerDelegationsModel"
}

func (s *StakerDelegationsModel) GetStateTransitions() (types.StateTransitions[StakerDelegationChange], []uint64) {
	stateChanges := make(types.StateTransitions[StakerDelegationChange])

	stateChanges[0] = func(log *storage.TransactionLog) (*StakerDelegationChange, error) {
		arguments, err := s.ParseLogArguments(log)
		if err != nil {
			return nil, err
		}

		delegated := true
		if log.EventName == "StakerUndelegated" {
			delegated = false
		}

		change := &StakerDelegationChange{
			Staker:           arguments[0].Value.(string),
			Operator:         arguments[1].Value.(string),
			Delegated:        delegated,
			TransactionHash:  log.TransactionHash,
			TransactionIndex: log.TransactionIndex,
			LogIndex:         log.LogIndex,
			BlockNumber:      log.BlockNumber,
			CreatedAt:        log.CreatedAt,
		}
		return change, nil
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

func (s *StakerDelegationsModel) HandleStateChange(log *storage.TransactionLog) (interface{}, error) {
	stateChanges, sortedBlockNumbers := s.GetStateTransitions()

	for _, blockNumber := range sortedBlockNumbers {
		if log.BlockNumber >= blockNumber {
			s.logger.Sugar().Debugw("Handling state change", zap.Uint64("blockNumber", blockNumber))

			change, err := stateChanges[blockNumber](log)
			if err != nil {
				return nil, err
			}

			if change != nil {
				wroteChange, err := s.writeStateChange(change)
				if err != nil {
					return wroteChange, err
				}
				return wroteChange, nil
			}
		}
	}
	return nil, nil
}

func (s *StakerDelegationsModel) writeStateChange(change *StakerDelegationChange) (interface{}, error) {
	s.logger.Sugar().Debugw("Writing state change", zap.Any("change", change))
	res := s.Db.Model(&StakerDelegationChange{}).Clauses(clause.Returning{}).Create(change)
	if res.Error != nil {
		s.logger.Error("Failed to insert into avs_operator_changes", zap.Error(res.Error))
		return change, res.Error
	}
	return change, nil
}

func (s *StakerDelegationsModel) WriteFinalState(blockNumber uint64) error {
	query := `
		with new_changes as (
			select
				staker,
				operator,
				block_number,
				max(transaction_index) as transaction_index,
				max(log_index) as log_index
			from staker_delegation_changes
			where block_number = @currentBlock
			group by 1, 2, 3
		),
		unique_delegations as (
			select
				nc.staker,
				nc.operator,
				sdc.log_index,
				sdc.delegated,
				nc.block_number
			from new_changes as nc
			left join staker_delegation_changes as sdc on (
				sdc.staker = nc.staker
				and sdc.operator = nc.operator
				and sdc.log_index = nc.log_index
				and sdc.transaction_index = nc.transaction_index
				and sdc.block_number = nc.block_number
			)
		),
		undelegations as (
			select
				concat(staker, '_', operator) as staker_operator
			from unique_delegations
			where delegated = false
		),
		carryover as (
			select
				rao.staker,
				rao.operator,
				@currentBlock as block_number
			from delegated_stakers as rao
			where
				rao.block_number = @previousBlock
				and concat(rao.staker, '_', rao.operator) not in (select staker_operator from undelegations)
		),
		final_state as (
			(select staker, operator, block_number::bigint from carryover)
			union all
			(select staker, operator, block_number::bigint from unique_delegations where delegated = true)
		)
		insert into delegated_stakers (staker, operator, block_number)
			select staker, operator, block_number from final_state
	`

	res := s.Db.Exec(query,
		sql.Named("currentBlock", blockNumber),
		sql.Named("previousBlock", blockNumber-1),
	)
	if res.Error != nil {
		s.logger.Sugar().Errorw("Failed to insert into operator_shares", zap.Error(res.Error))
		return res.Error
	}
	return nil
}
func (s *StakerDelegationsModel) getDifferenceInStates(blockNumber uint64) ([]DelegatedStakersDiff, error) {
	query := `
		with new_states as (
			select
				staker,
				operator,
				block_number,
				true as delegated
			from delegated_stakers
			where block_number = @currentBlock
		),
		previous_states as (
			select
				staker,
				operator,
				block_number,
				true as delegated
			from delegated_stakers
			where block_number = @previousBlock
		),
		undelegated as (
			(select staker, operator, delegated from previous_states)
			except
			(select staker, operator, delegated from new_states)
		),
		new_delegated as (
			(select staker, operator, delegated from new_states)
			except
			(select staker, operator, delegated from previous_states)
		)
		select staker, operator, false as delegated from undelegated
		union all
		select staker, operator, true as delegated from new_delegated;
	`
	results := make([]DelegatedStakersDiff, 0)
	res := s.Db.Model(&DelegatedStakersDiff{}).
		Raw(query,
			sql.Named("currentBlock", blockNumber),
			sql.Named("previousBlock", blockNumber-1),
		).
		Scan(&results)

	if res.Error != nil {
		s.logger.Sugar().Errorw("Failed to fetch delegated_stakers", zap.Error(res.Error))
		return nil, res.Error
	}
	return results, nil
}

func (s *StakerDelegationsModel) GenerateStateRoot(blockNumber uint64) (types.StateRoot, error) {
	results, err := s.getDifferenceInStates(blockNumber)
	if err != nil {
		return "", err
	}

	fullTree, err := s.merkelizeState(blockNumber, results)
	if err != nil {
		return "", err
	}
	return types.StateRoot(utils.ConvertBytesToString(fullTree.Root())), nil
}

func (s *StakerDelegationsModel) merkelizeState(blockNumber uint64, delegatedStakers []DelegatedStakersDiff) (*merkletree.MerkleTree, error) {
	// Operator -> staker:delegated
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
