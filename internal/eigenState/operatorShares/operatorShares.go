package operatorShares

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/Layr-Labs/go-sidecar/internal/config"
	"github.com/Layr-Labs/go-sidecar/internal/eigenState/base"
	"github.com/Layr-Labs/go-sidecar/internal/eigenState/stateManager"
	"github.com/Layr-Labs/go-sidecar/internal/eigenState/types"
	"github.com/Layr-Labs/go-sidecar/internal/storage"
	"github.com/Layr-Labs/go-sidecar/internal/types/numbers"
	"github.com/Layr-Labs/go-sidecar/internal/utils"
	"github.com/wealdtech/go-merkletree/v2"
	"github.com/wealdtech/go-merkletree/v2/keccak256"
	orderedmap "github.com/wk8/go-ordered-map/v2"
	"go.uber.org/zap"
	"golang.org/x/xerrors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"math/big"
	"slices"
	"sort"
	"strings"
	"time"
)

// OperatorShares represents the state of an operator's shares in a strategy at a given block number
type OperatorShares struct {
	Operator    string
	Strategy    string
	Shares      string `gorm:"type:numeric"`
	BlockNumber uint64
	CreatedAt   time.Time
}

// AccumulatedStateChange represents the accumulated state change for an operator's shares in a strategy at a given block number
type AccumulatedStateChange struct {
	Operator    string
	Strategy    string
	Shares      *big.Int
	BlockNumber uint64
	IsNegative  bool
}

type OperatorSharesDiff struct {
	Operator    string
	Strategy    string
	Shares      *big.Int
	BlockNumber uint64
	IsNew       bool
}

// SlotId is a unique identifier for an operator's shares in a strategy
type SlotId string

func NewSlotId(operator string, strategy string) SlotId {
	return SlotId(fmt.Sprintf("%s_%s", operator, strategy))
}

// Implements IEigenStateModel
type OperatorSharesModel struct {
	base.BaseEigenState
	StateTransitions types.StateTransitions[AccumulatedStateChange]
	Db               *gorm.DB
	Network          config.Network
	Environment      config.Environment
	logger           *zap.Logger
	globalConfig     *config.Config

	// Accumulates state changes for SlotIds, grouped by block number
	stateAccumulator map[uint64]map[SlotId]*AccumulatedStateChange
}

func NewOperatorSharesModel(
	esm *stateManager.EigenStateManager,
	grm *gorm.DB,
	Network config.Network,
	Environment config.Environment,
	logger *zap.Logger,
	globalConfig *config.Config,
) (*OperatorSharesModel, error) {
	model := &OperatorSharesModel{
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

	esm.RegisterState(model, 1)
	return model, nil
}

func (osm *OperatorSharesModel) GetModelName() string {
	return "OperatorSharesModel"
}

type operatorSharesOutput struct {
	Strategy string      `json:"strategy"`
	Shares   json.Number `json:"shares"`
}

func parseLogOutputForOperatorShares(outputDataStr string) (*operatorSharesOutput, error) {
	outputData := &operatorSharesOutput{}
	decoder := json.NewDecoder(strings.NewReader(outputDataStr))
	decoder.UseNumber()

	err := decoder.Decode(&outputData)
	if err != nil {
		return nil, err
	}
	outputData.Strategy = strings.ToLower(outputData.Strategy)
	return outputData, err
}

func (osm *OperatorSharesModel) GetStateTransitions() (types.StateTransitions[AccumulatedStateChange], []uint64) {
	stateChanges := make(types.StateTransitions[AccumulatedStateChange])

	stateChanges[0] = func(log *storage.TransactionLog) (*AccumulatedStateChange, error) {
		arguments, err := osm.ParseLogArguments(log)
		if err != nil {
			return nil, err
		}
		outputData, err := parseLogOutputForOperatorShares(log.OutputData)
		if err != nil {
			return nil, err
		}

		// Sanity check to make sure we've got an initialized accumulator map for the block
		if _, ok := osm.stateAccumulator[log.BlockNumber]; !ok {
			return nil, xerrors.Errorf("No state accumulator found for block %d", log.BlockNumber)
		}
		operator := strings.ToLower(arguments[0].Value.(string))

		sharesStr := outputData.Shares.String()
		shares, success := numbers.NewBig257().SetString(sharesStr, 10)
		if !success {
			osm.logger.Sugar().Errorw("Failed to convert shares to big.Int",
				zap.String("shares", sharesStr),
				zap.String("transactionHash", log.TransactionHash),
				zap.Uint64("transactionIndex", log.TransactionIndex),
				zap.Uint64("blockNumber", log.BlockNumber),
			)
			return nil, xerrors.Errorf("Failed to convert shares to big.Int: %s", sharesStr)
		}

		isNegative := false
		// All shares are emitted as ABS(shares), so we need to negate the shares if the event is a decrease
		if log.EventName == "OperatorSharesDecreased" {
			isNegative = true
		}

		slotId := NewSlotId(operator, outputData.Strategy)
		record, ok := osm.stateAccumulator[log.BlockNumber][slotId]
		if !ok {
			record = &AccumulatedStateChange{
				Operator:    operator,
				Strategy:    outputData.Strategy,
				Shares:      shares,
				BlockNumber: log.BlockNumber,
				IsNegative:  isNegative,
			}
			osm.stateAccumulator[log.BlockNumber][slotId] = record
		} else {
			if isNegative {
				record.Shares = record.Shares.Sub(record.Shares, shares)
			} else {
				record.Shares = record.Shares.Add(record.Shares, shares)
			}
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

func (osm *OperatorSharesModel) getContractAddressesForEnvironment() map[string][]string {
	contracts := osm.globalConfig.GetContractsMapForEnvAndNetwork()
	return map[string][]string{
		contracts.DelegationManager: []string{
			"OperatorSharesIncreased",
			"OperatorSharesDecreased",
		},
	}
}

func (osm *OperatorSharesModel) IsInterestingLog(log *storage.TransactionLog) bool {
	addresses := osm.getContractAddressesForEnvironment()
	return osm.BaseEigenState.IsInterestingLog(addresses, log)
}

func (osm *OperatorSharesModel) InitBlockProcessing(blockNumber uint64) error {
	osm.stateAccumulator[blockNumber] = make(map[SlotId]*AccumulatedStateChange)
	return nil
}

func (osm *OperatorSharesModel) HandleStateChange(log *storage.TransactionLog) (interface{}, error) {
	stateChanges, sortedBlockNumbers := osm.GetStateTransitions()

	for _, blockNumber := range sortedBlockNumbers {
		if log.BlockNumber >= blockNumber {
			osm.logger.Sugar().Debugw("Handling state change", zap.Uint64("blockNumber", blockNumber))

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

func (osm *OperatorSharesModel) clonePreviousBlocksToNewBlock(blockNumber uint64) error {
	query := `
		insert into operator_shares (operator, strategy, shares, block_number)
			select
				operator,
				strategy,
				shares,
				@currentBlock as block_number
			from operator_shares
			where block_number = @previousBlock
	`
	res := osm.Db.Exec(query,
		sql.Named("currentBlock", blockNumber),
		sql.Named("previousBlock", blockNumber-1),
	)

	if res.Error != nil {
		osm.logger.Sugar().Errorw("Failed to clone previous block state to new block", zap.Error(res.Error))
		return res.Error
	}
	return nil
}

// prepareState prepares the state for commit by adding the new state to the existing state
func (osm *OperatorSharesModel) prepareState(blockNumber uint64) ([]OperatorSharesDiff, error) {
	preparedState := make([]OperatorSharesDiff, 0)

	accumulatedState, ok := osm.stateAccumulator[blockNumber]
	if !ok {
		err := xerrors.Errorf("No accumulated state found for block %d", blockNumber)
		osm.logger.Sugar().Errorw(err.Error(), zap.Error(err), zap.Uint64("blockNumber", blockNumber))
		return nil, err
	}

	slotIds := make([]SlotId, 0)
	for slotId, _ := range accumulatedState {
		slotIds = append(slotIds, slotId)
	}

	// Find only the records from the previous block, that are modified in this block
	query := `
		select
			operator,
			strategy,
			shares
		from operator_shares
		where
			block_number = @previousBlock
			and concat(operator, '_', strategy) in @slotIds
	`
	existingRecords := make([]OperatorShares, 0)
	res := osm.Db.Model(&OperatorShares{}).
		Raw(query,
			sql.Named("previousBlock", blockNumber-1),
			sql.Named("slotIds", slotIds),
		).
		Scan(&existingRecords)

	if res.Error != nil {
		osm.logger.Sugar().Errorw("Failed to fetch operator_shares", zap.Error(res.Error))
		return nil, res.Error
	}

	// Map the existing records to a map for easier lookup
	mappedRecords := make(map[SlotId]OperatorShares)
	for _, record := range existingRecords {
		slotId := NewSlotId(record.Operator, record.Strategy)
		mappedRecords[slotId] = record
	}

	// Loop over our new state changes.
	// If the record exists in the previous block, add the shares to the existing shares
	for slotId, newState := range accumulatedState {
		prepared := OperatorSharesDiff{
			Operator:    newState.Operator,
			Strategy:    newState.Strategy,
			Shares:      newState.Shares,
			BlockNumber: blockNumber,
			IsNew:       false,
		}

		if existingRecord, ok := mappedRecords[slotId]; ok {
			existingShares, success := numbers.NewBig257().SetString(existingRecord.Shares, 10)
			if !success {
				osm.logger.Sugar().Errorw("Failed to convert existing shares to big.Int")
				continue
			}
			prepared.Shares = existingShares.Add(existingShares, newState.Shares)
		} else {
			// SlotID was not found in the previous block, so this is a new record
			prepared.IsNew = true
		}

		preparedState = append(preparedState, prepared)
	}
	return preparedState, nil
}

func (osm *OperatorSharesModel) CommitFinalState(blockNumber uint64) error {
	// Clone the previous block state to give us a reference point.
	err := osm.clonePreviousBlocksToNewBlock(blockNumber)
	if err != nil {
		return err
	}

	records, err := osm.prepareState(blockNumber)
	if err != nil {
		return err
	}

	newRecords := make([]OperatorShares, 0)
	updateRecords := make([]OperatorShares, 0)

	for _, record := range records {
		r := &OperatorShares{
			Operator:    record.Operator,
			Strategy:    record.Strategy,
			Shares:      record.Shares.String(),
			BlockNumber: record.BlockNumber,
		}
		if record.IsNew {
			newRecords = append(newRecords, *r)
		} else {
			updateRecords = append(updateRecords, *r)
		}
	}

	// Batch insert new records
	if len(newRecords) > 0 {
		res := osm.Db.Model(&OperatorShares{}).Clauses(clause.Returning{}).Create(&newRecords)
		if res.Error != nil {
			osm.logger.Sugar().Errorw("Failed to create new operator_shares records", zap.Error(res.Error))
			return res.Error
		}
	}
	// Update existing records that were cloned from the previous block
	if len(updateRecords) > 0 {
		for _, record := range updateRecords {
			res := osm.Db.Model(&OperatorShares{}).
				Where("operator = ? and strategy = ? and block_number = ?", record.Operator, record.Strategy, record.BlockNumber).
				Updates(map[string]interface{}{
					"shares": record.Shares,
				})
			if res.Error != nil {
				osm.logger.Sugar().Errorw("Failed to update operator_shares record", zap.Error(res.Error))
				return res.Error
			}
		}
	}

	return nil
}

func (osm *OperatorSharesModel) ClearAccumulatedState(blockNumber uint64) error {
	delete(osm.stateAccumulator, blockNumber)
	return nil
}

func (osm *OperatorSharesModel) GenerateStateRoot(blockNumber uint64) (types.StateRoot, error) {
	diffs, err := osm.prepareState(blockNumber)
	if err != nil {
		return "", err
	}

	fullTree, err := osm.merkelizeState(blockNumber, diffs)
	if err != nil {
		return "", err
	}
	return types.StateRoot(utils.ConvertBytesToString(fullTree.Root())), nil
}

func (osm *OperatorSharesModel) merkelizeState(blockNumber uint64, diffs []OperatorSharesDiff) (*merkletree.MerkleTree, error) {
	// Create a merkle tree with the structure:
	// strategy: map[operators]: shares
	om := orderedmap.New[string, *orderedmap.OrderedMap[string, string]]()

	for _, diff := range diffs {
		existingStrategy, found := om.Get(diff.Strategy)
		if !found {
			existingStrategy = orderedmap.New[string, string]()
			om.Set(diff.Strategy, existingStrategy)

			prev := om.GetPair(diff.Strategy).Prev()
			if prev != nil && strings.Compare(prev.Key, diff.Strategy) >= 0 {
				om.Delete(diff.Strategy)
				return nil, fmt.Errorf("strategy not in order")
			}
		}
		existingStrategy.Set(diff.Operator, diff.Shares.String())

		prev := existingStrategy.GetPair(diff.Operator).Prev()
		if prev != nil && strings.Compare(prev.Key, diff.Operator) >= 0 {
			existingStrategy.Delete(diff.Operator)
			return nil, fmt.Errorf("operator not in order")
		}
	}

	leaves := osm.InitializeMerkleTreeBaseStateWithBlock(blockNumber)
	for strat := om.Oldest(); strat != nil; strat = strat.Next() {

		operatorLeaves := make([][]byte, 0)
		for operator := strat.Value.Oldest(); operator != nil; operator = operator.Next() {
			operatorAddr := operator.Key
			shares := operator.Value
			operatorLeaves = append(operatorLeaves, encodeOperatorSharesLeaf(operatorAddr, shares))
		}

		stratTree, err := merkletree.NewTree(
			merkletree.WithData(operatorLeaves),
			merkletree.WithHashType(keccak256.New()),
		)
		if err != nil {
			return nil, err
		}
		leaves = append(leaves, encodeStratTree(strat.Key, stratTree.Root()))
	}
	return merkletree.NewTree(
		merkletree.WithData(leaves),
		merkletree.WithHashType(keccak256.New()),
	)
}

func encodeOperatorSharesLeaf(operator string, shares string) []byte {
	operatorBytes := []byte(operator)
	sharesBytes := []byte(shares)

	return append(operatorBytes, sharesBytes[:]...)
}

func encodeStratTree(strategy string, operatorTreeRoot []byte) []byte {
	strategyBytes := []byte(strategy)
	return append(strategyBytes, operatorTreeRoot[:]...)
}

func (osm *OperatorSharesModel) DeleteState(startBlockNumber uint64, endBlockNumber uint64) error {
	return osm.BaseEigenState.DeleteState("operator_shares", startBlockNumber, endBlockNumber, osm.Db)
}
