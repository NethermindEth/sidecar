package stakerShares

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

type StakerShares struct {
	Staker      string
	Strategy    string
	Shares      string `gorm:"type:numeric"`
	BlockNumber uint64
	CreatedAt   time.Time
}

type AccumulatedStateChange struct {
	Staker      string
	Strategy    string
	Shares      *big.Int
	BlockNumber uint64
	IsNegative  bool
}

type StakerSharesDiff struct {
	Staker      string
	Strategy    string
	Shares      *big.Int
	BlockNumber uint64
	IsNew       bool
}

type SlotId string

func NewSlotId(staker string, strategy string) SlotId {
	return SlotId(fmt.Sprintf("%s_%s", staker, strategy))
}

type StakerSharesModel struct {
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

func NewStakerSharesModel(
	esm *stateManager.EigenStateManager,
	grm *gorm.DB,
	network config.Network,
	environment config.Environment,
	logger *zap.Logger,
	globalConfig *config.Config,
) (*StakerSharesModel, error) {
	model := &StakerSharesModel{
		BaseEigenState:   base.BaseEigenState{},
		Db:               grm,
		Network:          network,
		Environment:      environment,
		logger:           logger,
		globalConfig:     globalConfig,
		stateAccumulator: make(map[uint64]map[SlotId]*AccumulatedStateChange),
	}

	esm.RegisterState(model, 3)
	return model, nil
}

func (ss *StakerSharesModel) GetModelName() string {
	return "StakerShares"
}

type depositOutputData struct {
	Depositor string      `json:"depositor"`
	Staker    string      `json:"staker"`
	Strategy  string      `json:"strategy"`
	Shares    json.Number `json:"shares"`
}

// parseLogOutputForDepositEvent parses the output data of a Deposit event
// Custom parser to preserve the precision of the shares value.
// Allowing the standard json.Unmarshal to parse the shares value to a float64 which
// causes it to lose precision by being represented as scientific notation.
func parseLogOutputForDepositEvent(outputDataStr string) (*depositOutputData, error) {
	outputData := &depositOutputData{}
	decoder := json.NewDecoder(strings.NewReader(outputDataStr))
	decoder.UseNumber()

	err := decoder.Decode(&outputData)
	if err != nil {
		return nil, err
	}
	outputData.Staker = strings.ToLower(outputData.Staker)
	outputData.Depositor = strings.ToLower(outputData.Depositor)
	outputData.Strategy = strings.ToLower(outputData.Strategy)
	return outputData, err
}

func (ss *StakerSharesModel) handleStakerDepositEvent(log *storage.TransactionLog) (*AccumulatedStateChange, error) {
	outputData, err := parseLogOutputForDepositEvent(log.OutputData)
	if err != nil {
		return nil, err
	}

	var stakerAddress string
	if outputData.Depositor != "" {
		stakerAddress = outputData.Depositor
	}
	if outputData.Staker != "" {
		stakerAddress = outputData.Staker
	}

	if stakerAddress == "" {
		return nil, xerrors.Errorf("No staker address found in event")
	}

	shares, success := numbers.NewBig257().SetString(outputData.Shares.String(), 10)
	if !success {
		return nil, xerrors.Errorf("Failed to convert shares to big.Int: %s", outputData.Shares)
	}

	return &AccumulatedStateChange{
		Staker:      stakerAddress,
		Strategy:    outputData.Strategy,
		Shares:      shares,
		BlockNumber: log.BlockNumber,
	}, nil
}

type podSharesUpdatedOutputData struct {
	SharesDelta json.Number `json:"sharesDelta"`
}

func parseLogOutputForPodSharesUpdatedEvent(outputDataStr string) (*podSharesUpdatedOutputData, error) {
	outputData := &podSharesUpdatedOutputData{}
	decoder := json.NewDecoder(strings.NewReader(outputDataStr))
	decoder.UseNumber()

	err := decoder.Decode(&outputData)
	if err != nil {
		return nil, err
	}
	return outputData, err
}

func (ss *StakerSharesModel) handlePodSharesUpdatedEvent(log *storage.TransactionLog) (*AccumulatedStateChange, error) {
	arguments, err := ss.ParseLogArguments(log)
	if err != nil {
		return nil, err
	}
	outputData, err := parseLogOutputForPodSharesUpdatedEvent(log.OutputData)
	if err != nil {
		return nil, err
	}

	staker := strings.ToLower(arguments[0].Value.(string))

	sharesDeltaStr := outputData.SharesDelta.String()

	sharesDelta, success := numbers.NewBig257().SetString(sharesDeltaStr, 10)
	if !success {
		return nil, xerrors.Errorf("Failed to convert shares to big.Int: %s", sharesDelta)
	}

	return &AccumulatedStateChange{
		Staker:      staker,
		Strategy:    "0xbeac0eeeeeeeeeeeeeeeeeeeeeeeeeeeeeebeac0",
		Shares:      sharesDelta,
		BlockNumber: log.BlockNumber,
	}, nil
}

func (ss *StakerSharesModel) handleM1StakerWithdrawals(log *storage.TransactionLog) (*AccumulatedStateChange, error) {
	outputData, err := parseLogOutputForDepositEvent(log.OutputData)
	if err != nil {
		return nil, err
	}

	var stakerAddress string
	if outputData.Depositor != "" {
		stakerAddress = outputData.Depositor
	}
	if outputData.Staker != "" {
		stakerAddress = outputData.Staker
	}

	if stakerAddress == "" {
		return nil, xerrors.Errorf("No staker address found in event")
	}

	shares, success := numbers.NewBig257().SetString(outputData.Shares.String(), 10)
	if !success {
		return nil, xerrors.Errorf("Failed to convert shares to big.Int: %s", outputData.Shares)
	}

	return &AccumulatedStateChange{
		Staker:      stakerAddress,
		Strategy:    outputData.Strategy,
		Shares:      shares,
		BlockNumber: log.BlockNumber,
		IsNegative:  true,
	}, nil
}

func (ss *StakerSharesModel) handleM2StakerWithdrawals(log *storage.TransactionLog) (*AccumulatedStateChange, error) {
	// TODO(seanmcgary): come back to this...
	return nil, nil
}

func (ss *StakerSharesModel) GetStateTransitions() (types.StateTransitions[AccumulatedStateChange], []uint64) {
	stateChanges := make(types.StateTransitions[AccumulatedStateChange])

	stateChanges[0] = func(log *storage.TransactionLog) (*AccumulatedStateChange, error) {
		var parsedRecord *AccumulatedStateChange
		var err error

		contractAddresses := ss.globalConfig.GetContractsMapForEnvAndNetwork()

		// Staker shares is a bit more complex and has 4 possible contract/event combinations
		// that we need to handle
		if log.Address == contractAddresses.StrategyManager && log.EventName == "Deposit" {
			parsedRecord, err = ss.handleStakerDepositEvent(log)
		} else if log.Address == contractAddresses.EigenpodManager && log.EventName == "PodSharesUpdated" {
			parsedRecord, err = ss.handlePodSharesUpdatedEvent(log)
		} else if log.Address == contractAddresses.StrategyManager && log.EventName == "ShareWithdrawalQueued" && log.TransactionHash != "0x62eb0d0865b2636c74ed146e2d161e39e42b09bac7f86b8905fc7a830935dc1e" {
			parsedRecord, err = ss.handleM1StakerWithdrawals(log)
		} else if log.Address == contractAddresses.DelegationManager && log.EventName == "WithdrawalMigrated" {
			parsedRecord, err = ss.handleM2StakerWithdrawals(log)
		} else {
			ss.logger.Sugar().Debugw("Got stakerShares event that we don't handle",
				zap.String("eventName", log.EventName),
				zap.String("address", log.Address),
			)
		}
		if err != nil {
			return nil, err
		}
		if parsedRecord == nil {
			return nil, nil
		}

		// Sanity check to make sure we've got an initialized accumulator map for the block
		if _, ok := ss.stateAccumulator[log.BlockNumber]; !ok {
			return nil, xerrors.Errorf("No state accumulator found for block %d", log.BlockNumber)
		}

		slotId := NewSlotId(parsedRecord.Staker, parsedRecord.Strategy)
		record, ok := ss.stateAccumulator[log.BlockNumber][slotId]
		if !ok {
			record = parsedRecord
			ss.stateAccumulator[log.BlockNumber][slotId] = record
		} else {
			if record.IsNegative {
				record.Shares = record.Shares.Sub(record.Shares, parsedRecord.Shares)
			} else {
				record.Shares = record.Shares.Add(record.Shares, parsedRecord.Shares)
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

func (ss *StakerSharesModel) getContractAddressesForEnvironment() map[string][]string {
	contracts := ss.globalConfig.GetContractsMapForEnvAndNetwork()
	return map[string][]string{
		contracts.DelegationManager: []string{
			"WithdrawalMigrated",
		},
		contracts.StrategyManager: []string{
			"Deposit",
			"ShareWithdrawalQueued",
		},
		contracts.EigenpodManager: []string{
			"PodSharesUpdated",
		},
	}
}

func (ss *StakerSharesModel) IsInterestingLog(log *storage.TransactionLog) bool {
	addresses := ss.getContractAddressesForEnvironment()
	return ss.BaseEigenState.IsInterestingLog(addresses, log)
}

func (ss *StakerSharesModel) InitBlockProcessing(blockNumber uint64) error {
	ss.stateAccumulator[blockNumber] = make(map[SlotId]*AccumulatedStateChange)
	return nil
}

func (ss *StakerSharesModel) HandleStateChange(log *storage.TransactionLog) (interface{}, error) {
	stateChanges, sortedBlockNumbers := ss.GetStateTransitions()

	for _, blockNumber := range sortedBlockNumbers {
		if log.BlockNumber >= blockNumber {
			ss.logger.Sugar().Debugw("Handling state change", zap.Uint64("blockNumber", blockNumber))

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

func (ss *StakerSharesModel) clonePreviousBlocksToNewBlock(blockNumber uint64) error {
	query := `
		insert into staker_shares (staker, strategy, shares, block_number)
			select
				staker,
				strategy,
				shares,
				@currentBlock as block_number
			from staker_shares
			where block_number = @previousBlock
	`
	res := ss.Db.Exec(query,
		sql.Named("currentBlock", blockNumber),
		sql.Named("previousBlock", blockNumber-1),
	)

	if res.Error != nil {
		ss.logger.Sugar().Errorw("Failed to clone previous block state to new block", zap.Error(res.Error))
		return res.Error
	}
	return nil
}

// prepareState prepares the state for commit by adding the new state to the existing state
func (ss *StakerSharesModel) prepareState(blockNumber uint64) ([]StakerSharesDiff, error) {
	preparedState := make([]StakerSharesDiff, 0)

	accumulatedState, ok := ss.stateAccumulator[blockNumber]
	if !ok {
		err := xerrors.Errorf("No accumulated state found for block %d", blockNumber)
		ss.logger.Sugar().Errorw(err.Error(), zap.Error(err), zap.Uint64("blockNumber", blockNumber))
		return nil, err
	}

	slotIds := make([]SlotId, 0)
	for slotId, _ := range accumulatedState {
		slotIds = append(slotIds, slotId)
	}

	// Find only the records from the previous block, that are modified in this block
	query := `
		select
			staker,
			strategy,
			shares
		from staker_shares
		where
			block_number = @previousBlock
			and concat(staker, '_', strategy) in @slotIds
	`
	existingRecords := make([]StakerShares, 0)
	res := ss.Db.Model(&StakerShares{}).
		Raw(query,
			sql.Named("previousBlock", blockNumber-1),
			sql.Named("slotIds", slotIds),
		).
		Scan(&existingRecords)

	if res.Error != nil {
		ss.logger.Sugar().Errorw("Failed to fetch staker_shares", zap.Error(res.Error))
		return nil, res.Error
	}

	// Map the existing records to a map for easier lookup
	mappedRecords := make(map[SlotId]StakerShares)
	for _, record := range existingRecords {
		slotId := NewSlotId(record.Staker, record.Strategy)
		mappedRecords[slotId] = record
	}

	// Loop over our new state changes.
	// If the record exists in the previous block, add the shares to the existing shares
	for slotId, newState := range accumulatedState {
		prepared := StakerSharesDiff{
			Staker:      newState.Staker,
			Strategy:    newState.Strategy,
			Shares:      newState.Shares,
			BlockNumber: blockNumber,
			IsNew:       false,
		}

		if existingRecord, ok := mappedRecords[slotId]; ok {
			existingShares, success := numbers.NewBig257().SetString(existingRecord.Shares, 10)
			if !success {
				ss.logger.Sugar().Errorw("Failed to convert existing shares to big.Int")
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

func (ss *StakerSharesModel) CommitFinalState(blockNumber uint64) error {
	// Clone the previous block state to give us a reference point.
	err := ss.clonePreviousBlocksToNewBlock(blockNumber)
	if err != nil {
		return err
	}

	records, err := ss.prepareState(blockNumber)
	if err != nil {
		return err
	}

	newRecords := make([]StakerShares, 0)
	updateRecords := make([]StakerShares, 0)

	for _, record := range records {
		r := &StakerShares{
			Staker:      record.Staker,
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
		res := ss.Db.Model(&StakerShares{}).Clauses(clause.Returning{}).Create(&newRecords)
		if res.Error != nil {
			ss.logger.Sugar().Errorw("Failed to create new operator_shares records", zap.Error(res.Error))
			return res.Error
		}
	}
	// Update existing records that were cloned from the previous block
	if len(updateRecords) > 0 {
		for _, record := range updateRecords {
			res := ss.Db.Model(&StakerShares{}).
				Where("staker = ? and strategy = ? and block_number = ?", record.Staker, record.Strategy, record.BlockNumber).
				Updates(map[string]interface{}{
					"shares": record.Shares,
				})
			if res.Error != nil {
				ss.logger.Sugar().Errorw("Failed to update operator_shares record", zap.Error(res.Error))
				return res.Error
			}
		}
	}

	return nil
}

func (ss *StakerSharesModel) ClearAccumulatedState(blockNumber uint64) error {
	delete(ss.stateAccumulator, blockNumber)
	return nil
}

func (ss *StakerSharesModel) GenerateStateRoot(blockNumber uint64) (types.StateRoot, error) {
	diffs, err := ss.prepareState(blockNumber)
	if err != nil {
		return "", err
	}

	fullTree, err := ss.merkelizeState(blockNumber, diffs)
	if err != nil {
		return "", err
	}
	return types.StateRoot(utils.ConvertBytesToString(fullTree.Root())), nil
}

func (ss *StakerSharesModel) merkelizeState(blockNumber uint64, diffs []StakerSharesDiff) (*merkletree.MerkleTree, error) {
	// Create a merkle tree with the structure:
	// strategy: map[staker]: shares
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
		existingStrategy.Set(diff.Staker, diff.Shares.String())

		prev := existingStrategy.GetPair(diff.Staker).Prev()
		if prev != nil && strings.Compare(prev.Key, diff.Staker) >= 0 {
			existingStrategy.Delete(diff.Staker)
			return nil, fmt.Errorf("operator not in order")
		}
	}

	leaves := ss.InitializeMerkleTreeBaseStateWithBlock(blockNumber)
	for strat := om.Oldest(); strat != nil; strat = strat.Next() {

		stakerLeaves := make([][]byte, 0)
		for staker := strat.Value.Oldest(); staker != nil; staker = staker.Next() {
			stakerAddr := staker.Key
			shares := staker.Value
			stakerLeaves = append(stakerLeaves, encodeStakerSharesLeaf(stakerAddr, shares))
		}

		stratTree, err := merkletree.NewTree(
			merkletree.WithData(stakerLeaves),
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

func encodeStakerSharesLeaf(staker string, shares string) []byte {
	stakerBytes := []byte(staker)
	sharesBytes := []byte(shares)

	return append(stakerBytes, sharesBytes[:]...)
}

func encodeStratTree(strategy string, stakerTreeRoot []byte) []byte {
	strategyBytes := []byte(strategy)
	return append(strategyBytes, stakerTreeRoot[:]...)
}

func (ss *StakerSharesModel) DeleteState(startBlockNumber uint64, endBlockNumber uint64) error {
	return ss.BaseEigenState.DeleteState("staker_shares", startBlockNumber, endBlockNumber, ss.Db)
}
