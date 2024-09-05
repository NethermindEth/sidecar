package eigenState

import (
	"database/sql"
	"fmt"
	"github.com/Layr-Labs/sidecar/internal/config"
	"github.com/Layr-Labs/sidecar/internal/storage"
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
	BaseEigenState
	StateTransitions StateTransitions[AvsOperatorChange]
	Db               *gorm.DB
	Network          config.Network
	Environment      config.Environment
	logger           *zap.Logger
	globalConfig     *config.Config
}

// Create new instance of AvsOperators state model
func NewAvsOperators(
	esm *EigenStateManager,
	grm *gorm.DB,
	Network config.Network,
	Environment config.Environment,
	logger *zap.Logger,
	globalConfig *config.Config,
) (*AvsOperators, error) {
	s := &AvsOperators{
		BaseEigenState: BaseEigenState{},
		Db:             grm,
		Network:        Network,
		Environment:    Environment,
		logger:         logger,
		globalConfig:   globalConfig,
	}
	esm.RegisterState(s)
	return s, nil
}

// Get the state transitions for the AvsOperators state model
//
// Each state transition is function indexed by a block number.
// BlockNumber 0 is the catchall state
//
// Returns the map and a reverse sorted list of block numbers that can be traversed when
// processing a log to determine which state change to apply.
func (a *AvsOperators) GetStateTransitions() (StateTransitions[AvsOperatorChange], []uint64) {
	stateChanges := make(StateTransitions[AvsOperatorChange])

	// TODO(seanmcgary): make this not a closure so this function doesnt get big an messy...
	stateChanges[0] = func(log *storage.TransactionLog) (*AvsOperatorChange, error) {
		// TODO(seanmcgary): actually parse the log
		change := &AvsOperatorChange{
			Operator:         "operator",
			Avs:              "avs",
			Registered:       true,
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
	logAddress := strings.ToLower(log.Address)
	if eventNames, ok := addresses[logAddress]; ok {
		if slices.Contains(eventNames, log.EventName) {
			return true
		}
	}
	return false
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
func (a *AvsOperators) WriteFinalState(blockNumber uint64) error {
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

// Generates a state root for the given block number.
//
// 1. Select all registered_avs_operators for the given block number ordered by avs and operator asc
// 2. Create an ordered map, with AVSs at the top level that point to an ordered map of operators and block numbers
// 3. Create a merkle tree for each AVS, with the operator:block_number pairs as leaves
// 4. Create a merkle tree for all AVS trees
// 5. Return the root of the full tree
func (a *AvsOperators) GenerateStateRoot(blockNumber uint64) (StateRoot, error) {
	query := `
		select
			avs,
			operator,
			block_number
		from registered_avs_operators
		where
			block_number = @blockNumber
		order by avs asc, operator asc
	`
	results := make([]RegisteredAvsOperators, 0)
	res := a.Db.Model(&results).Raw(query, sql.Named("blockNumber", blockNumber))

	if res.Error != nil {
		a.logger.Sugar().Errorw("Failed to fetch registered_avs_operators", zap.Error(res.Error))
		return "", res.Error
	}

	// Avs -> operator:block_number
	om := orderedmap.New[string, *orderedmap.OrderedMap[string, uint64]]()

	for _, result := range results {
		existingAvs, found := om.Get(result.Avs)
		if !found {
			existingAvs = orderedmap.New[string, uint64]()
			om.Set(result.Avs, existingAvs)

			prev := om.GetPair(result.Avs).Prev()
			if prev != nil && strings.Compare(prev.Key, result.Avs) >= 0 {
				om.Delete(result.Avs)
				return "", fmt.Errorf("avs not in order")
			}
		}
		existingAvs.Set(result.Operator, result.BlockNumber)

		prev := existingAvs.GetPair(result.Operator).Prev()
		if prev != nil && strings.Compare(prev.Key, result.Operator) >= 0 {
			existingAvs.Delete(result.Operator)
			return "", fmt.Errorf("operator not in order")
		}
	}

	avsLeaves := make([][]byte, 0)
	for avs := om.Oldest(); avs != nil; avs = avs.Next() {

		operatorLeafs := make([][]byte, 0)
		for operator := avs.Value.Oldest(); operator != nil; operator = operator.Next() {
			operatorAddr := operator.Key
			block := operator.Value
			operatorLeafs = append(operatorLeafs, []byte(fmt.Sprintf("%s:%d", operatorAddr, block)))
		}

		avsTree, err := merkletree.NewTree(
			merkletree.WithData(operatorLeafs),
			merkletree.WithHashType(keccak256.New()),
		)
		if err != nil {
			return "", err
		}

		avsBytes := []byte(avs.Key)
		root := avsTree.Root()
		avsLeaves = append(avsLeaves, append(avsBytes, root[:]...))
	}

	fullTree, err := merkletree.NewTree(
		merkletree.WithData(avsLeaves),
		merkletree.WithHashType(keccak256.New()),
	)
	if err != nil {
		return "", err
	}
	return StateRoot(fullTree.Root()), nil
}
