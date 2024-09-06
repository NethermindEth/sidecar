package avsOperators

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

// Schema for registered_avs_operators block state table
type RegisteredAvsOperators struct {
	Operator    string
	Avs         string
	BlockNumber uint64
	CreatedAt   time.Time
}

// Schema for avs_operator_changes table
type AvsOperatorChange struct {
	Id               uint64 `gorm:"type:serial"`
	Operator         string
	Avs              string
	Registered       bool
	TransactionHash  string
	TransactionIndex uint64
	LogIndex         uint64
	BlockNumber      uint64
	CreatedAt        time.Time
}

// EigenState model for AVS operators that implements IEigenStateModel
type AvsOperators struct {
	base.BaseEigenState
	StateTransitions types.StateTransitions[AvsOperatorChange]
	Db               *gorm.DB
	Network          config.Network
	Environment      config.Environment
	logger           *zap.Logger
	globalConfig     *config.Config
}

type RegisteredAvsOperatorDiff struct {
	Operator    string
	Avs         string
	BlockNumber uint64
	Registered  bool
}

// Create new instance of AvsOperators state model
func NewAvsOperators(
	esm *stateManager.EigenStateManager,
	grm *gorm.DB,
	Network config.Network,
	Environment config.Environment,
	logger *zap.Logger,
	globalConfig *config.Config,
) (*AvsOperators, error) {
	s := &AvsOperators{
		BaseEigenState: base.BaseEigenState{
			Logger: logger,
		},
		Db:           grm,
		Network:      Network,
		Environment:  Environment,
		logger:       logger,
		globalConfig: globalConfig,
	}
	esm.RegisterState(s, 0)
	return s, nil
}

func (a *AvsOperators) GetModelName() string {
	return "AvsOperators"
}

// Get the state transitions for the AvsOperators state model
//
// Each state transition is function indexed by a block number.
// BlockNumber 0 is the catchall state
//
// Returns the map and a reverse sorted list of block numbers that can be traversed when
// processing a log to determine which state change to apply.
func (a *AvsOperators) GetStateTransitions() (types.StateTransitions[AvsOperatorChange], []uint64) {
	stateChanges := make(types.StateTransitions[AvsOperatorChange])

	// TODO(seanmcgary): make this not a closure so this function doesnt get big an messy...
	stateChanges[0] = func(log *storage.TransactionLog) (*AvsOperatorChange, error) {
		arguments, err := a.ParseLogArguments(log)
		if err != nil {
			return nil, err
		}

		outputData, err := a.ParseLogOutput(log)
		if err != nil {
			return nil, err
		}

		registered := false
		if val, ok := outputData["status"]; ok {
			registered = uint64(val.(float64)) == 1
		}

		change := &AvsOperatorChange{
			Operator:         arguments[0].Value.(string),
			Avs:              arguments[1].Value.(string),
			Registered:       registered,
			TransactionHash:  log.TransactionHash,
			TransactionIndex: log.TransactionIndex,
			LogIndex:         log.LogIndex,
			BlockNumber:      log.BlockNumber,
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

// Returns a map of contract addresses to event names that are interesting to the state model
func (a *AvsOperators) getContractAddressesForEnvironment() map[string][]string {
	contracts := a.globalConfig.GetContractsMapForEnvAndNetwork()
	return map[string][]string{
		contracts.AvsDirectory: []string{
			"OperatorAVSRegistrationStatusUpdated",
		},
	}
}

// Given a log, determine if it is interesting to the state model
func (a *AvsOperators) IsInterestingLog(log *storage.TransactionLog) bool {
	addresses := a.getContractAddressesForEnvironment()
	return a.BaseEigenState.IsInterestingLog(addresses, log)
}

func (a *AvsOperators) StartBlockProcessing(blockNumber uint64) error {
	return nil
}

// Handle the state change for the given log
//
// Takes a log and iterates over the state transitions to determine which state change to apply based on block number.
func (a *AvsOperators) HandleStateChange(log *storage.TransactionLog) (interface{}, error) {
	stateChanges, sortedBlockNumbers := a.GetStateTransitions()

	for _, blockNumber := range sortedBlockNumbers {
		if log.BlockNumber >= blockNumber {
			a.logger.Sugar().Debugw("Handling state change", zap.Uint64("blockNumber", blockNumber))

			change, err := stateChanges[blockNumber](log)
			if err != nil {
				return nil, err
			}

			if change != nil {
				wroteChange, err := a.writeStateChange(change)
				if err != nil {
					return wroteChange, err
				}
				return wroteChange, nil
			}
		}
	}
	return nil, nil
}

// Write the state change to the database
func (a *AvsOperators) writeStateChange(change *AvsOperatorChange) (*AvsOperatorChange, error) {
	a.logger.Sugar().Debugw("Writing state change", zap.Any("change", change))
	res := a.Db.Model(&AvsOperatorChange{}).Clauses(clause.Returning{}).Create(change)
	if res.Error != nil {
		a.logger.Error("Failed to insert into avs_operator_changes", zap.Error(res.Error))
		return change, res.Error
	}
	return change, nil
}

// Write the new final state to the database.
//
// 1. Get latest distinct change value for each avs/operator
// 2. Join the latest unique change value with the previous blocks state to overlay new changes
// 3. Filter joined set on registered = false to get unregistrations
// 4. Determine which rows from the previous block should be carried over and which shouldnt (i.e. deregistrations)
// 5. Geneate the final state by unioning the carryover and the new registrations
// 6. Insert the final state into the registered_avs_operators table
func (a *AvsOperators) CommitFinalState(blockNumber uint64) error {
	query := `
		with new_changes as (
			select
				avs,
				operator,
				block_number,
				max(transaction_index) as transaction_index,
				max(log_index) as log_index
			from avs_operator_changes
			where block_number = @currentBlock
			group by 1, 2, 3
		),
		unique_registrations as (
			select
				nc.avs,
				nc.operator,
				aoc.log_index,
				aoc.registered,
				nc.block_number
			from new_changes as nc
			left join avs_operator_changes as aoc on (
				aoc.avs = nc.avs
				and aoc.operator = nc.operator
				and aoc.log_index = nc.log_index
				and aoc.transaction_index = nc.transaction_index
				and aoc.block_number = nc.block_number
			)
		),
		unregistrations as (
			select
				concat(avs, '_', operator) as operator_avs
			from unique_registrations
			where registered = false
		),
		carryover as (
			select
				rao.avs,
				rao.operator,
				@currentBlock as block_number
			from registered_avs_operators as rao
			where
				rao.block_number = @previousBlock
				and concat(rao.avs, '_', rao.operator) not in (select operator_avs from unregistrations)
		),
		final_state as (
			(select avs, operator, block_number::bigint from carryover)
			union all
			(select avs, operator, block_number::bigint from unique_registrations where registered = true)
		)
		insert into registered_avs_operators (avs, operator, block_number)
			select avs, operator, block_number from final_state
	`

	res := a.Db.Exec(query,
		sql.Named("currentBlock", blockNumber),
		sql.Named("previousBlock", blockNumber-1),
	)
	if res.Error != nil {
		a.logger.Sugar().Errorw("Failed to insert into registered_avs_operators", zap.Error(res.Error))
		return res.Error
	}
	return nil
}

func (a *AvsOperators) getDifferenceInStates(blockNumber uint64) ([]RegisteredAvsOperatorDiff, error) {
	query := `
		with new_states as (
			select
				avs,
				operator,
				block_number,
				true as registered
			from registered_avs_operators
			where block_number = @currentBlock
		),
		previous_states as (
			select
				avs,
				operator,
				block_number,
				true as registered
			from registered_avs_operators
			where block_number = @previousBlock
		),
		unregistered as (
			(select avs, operator, registered from previous_states)
			except
			(select avs, operator, registered from new_states)
		),
		new_registered as (
			(select avs, operator, registered from new_states)
			except
			(select avs, operator, registered from previous_states)
		)
		select avs, operator, false as registered from unregistered
		union all
		select avs, operator, true as registered from new_registered;
	`
	results := make([]RegisteredAvsOperatorDiff, 0)
	res := a.Db.Model(&RegisteredAvsOperatorDiff{}).
		Raw(query,
			sql.Named("currentBlock", blockNumber),
			sql.Named("previousBlock", blockNumber-1),
		).
		Scan(&results)

	if res.Error != nil {
		a.logger.Sugar().Errorw("Failed to fetch registered_avs_operators", zap.Error(res.Error))
		return nil, res.Error
	}
	return results, nil
}

func (a *AvsOperators) ClearAccumulatedState(blockNumber uint64) error {
	panic("implement me")
}

// Generates a state root for the given block number.
//
// 1. Select all registered_avs_operators for the given block number ordered by avs and operator asc
// 2. Create an ordered map, with AVSs at the top level that point to an ordered map of operators and block numbers
// 3. Create a merkle tree for each AVS, with the operator:block_number pairs as leaves
// 4. Create a merkle tree for all AVS trees
// 5. Return the root of the full tree
func (a *AvsOperators) GenerateStateRoot(blockNumber uint64) (types.StateRoot, error) {
	results, err := a.getDifferenceInStates(blockNumber)
	if err != nil {
		return "", err
	}

	fullTree, err := a.merkelizeState(blockNumber, results)
	if err != nil {
		return "", err
	}
	return types.StateRoot(utils.ConvertBytesToString(fullTree.Root())), nil
}

func (a *AvsOperators) merkelizeState(blockNumber uint64, avsOperators []RegisteredAvsOperatorDiff) (*merkletree.MerkleTree, error) {
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

func encodeOperatorLeaf(operator string, registered bool) []byte {
	return []byte(fmt.Sprintf("%s:%t", operator, registered))
}

func encodeAvsLeaf(avs string, avsOperatorRoot []byte) []byte {
	return append([]byte(avs), avsOperatorRoot[:]...)
}
