package rewardSubmissions

import (
	"database/sql"
	"encoding/json"
	"fmt"
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
	"slices"
	"sort"
	"strings"
	"time"
)

type RewardSubmission struct {
	Avs            string
	RewardHash     string
	Token          string
	Amount         string
	Strategy       string
	StrategyIndex  uint64
	Multiplier     string     `gorm:"type:numeric"`
	StartTimestamp *time.Time `gorm:"type:DATETIME"`
	EndTimestamp   *time.Time `gorm:"type:DATETIME"`
	Duration       uint64
	BlockNumber    uint64
	IsForAll       bool
}

type RewardSubmissionDiff struct {
	RewardSubmission *RewardSubmission
	IsNew            bool
	IsNoLongerActive bool
}

type RewardSubmissions struct {
	Submissions []*RewardSubmission
}

type SlotId string

func NewSlotId(rewardHash string, strategy string) SlotId {
	return SlotId(fmt.Sprintf("%s_%s", rewardHash, strategy))
}

type RewardSubmissionsModel struct {
	base.BaseEigenState
	StateTransitions types.StateTransitions[RewardSubmissions]
	Db               *gorm.DB
	Network          config.Network
	Environment      config.Environment
	logger           *zap.Logger
	globalConfig     *config.Config

	// Accumulates state changes for SlotIds, grouped by block number
	stateAccumulator map[uint64]map[SlotId]*RewardSubmission
}

func NewRewardSubmissionsModel(
	esm *stateManager.EigenStateManager,
	grm *gorm.DB,
	Network config.Network,
	Environment config.Environment,
	logger *zap.Logger,
	globalConfig *config.Config,
) (*RewardSubmissionsModel, error) {
	model := &RewardSubmissionsModel{
		BaseEigenState: base.BaseEigenState{
			Logger: logger,
		},
		Db:               grm,
		Network:          Network,
		Environment:      Environment,
		logger:           logger,
		globalConfig:     globalConfig,
		stateAccumulator: make(map[uint64]map[SlotId]*RewardSubmission),
	}

	esm.RegisterState(model, 4)
	return model, nil
}

func (rs *RewardSubmissionsModel) GetModelName() string {
	return "RewardSubmissionsModel"
}

type genericRewardPaymentData struct {
	Token                    string
	Amount                   json.Number
	StartTimestamp           uint64
	Duration                 uint64
	StrategiesAndMultipliers []struct {
		Strategy   string
		Multiplier json.Number
	} `json:"strategiesAndMultipliers"`
}

type rewardSubmissionOutputData struct {
	RewardsSubmission *genericRewardPaymentData `json:"rewardsSubmission"`
	RangePayment      *genericRewardPaymentData `json:"rangePayment"`
}

func parseRewardSubmissionOutputData(outputDataStr string) (*rewardSubmissionOutputData, error) {
	outputData := &rewardSubmissionOutputData{}
	decoder := json.NewDecoder(strings.NewReader(outputDataStr))
	decoder.UseNumber()

	err := decoder.Decode(&outputData)
	if err != nil {
		return nil, err
	}
	return outputData, err
}

func (rs *RewardSubmissionsModel) handleRewardSubmissionCreatedEvent(log *storage.TransactionLog) (*RewardSubmissions, error) {
	arguments, err := rs.ParseLogArguments(log)
	if err != nil {
		return nil, err
	}

	outputData, err := parseRewardSubmissionOutputData(log.OutputData)
	if err != nil {
		return nil, err
	}

	var actualOuputData *genericRewardPaymentData
	if log.EventName == "RangePaymentCreated" || log.EventName == "RangePaymentForAllCreated" {
		actualOuputData = outputData.RangePayment
	} else {
		actualOuputData = outputData.RewardsSubmission
	}

	rewardSubmissions := make([]*RewardSubmission, 0)

	for _, strategyAndMultiplier := range actualOuputData.StrategiesAndMultipliers {

		startTimestamp := time.Unix(int64(actualOuputData.StartTimestamp), 0)
		endTimestamp := startTimestamp.Add(time.Duration(actualOuputData.Duration) * time.Second)

		rewardSubmission := &RewardSubmission{
			Avs:            strings.ToLower(arguments[0].Value.(string)),
			RewardHash:     strings.ToLower(arguments[2].Value.(string)),
			Token:          strings.ToLower(actualOuputData.Token),
			Amount:         actualOuputData.Amount.String(),
			Strategy:       strategyAndMultiplier.Strategy,
			Multiplier:     strategyAndMultiplier.Multiplier.String(),
			StartTimestamp: &startTimestamp,
			EndTimestamp:   &endTimestamp,
			Duration:       actualOuputData.Duration,
			BlockNumber:    log.BlockNumber,
			IsForAll:       log.EventName == "RewardsSubmissionForAllCreated" || log.EventName == "RangePaymentForAllCreated",
		}
		rewardSubmissions = append(rewardSubmissions, rewardSubmission)
	}

	return &RewardSubmissions{Submissions: rewardSubmissions}, nil
}

func (rs *RewardSubmissionsModel) GetStateTransitions() (types.StateTransitions[RewardSubmissions], []uint64) {
	stateChanges := make(types.StateTransitions[RewardSubmissions])

	stateChanges[0] = func(log *storage.TransactionLog) (*RewardSubmissions, error) {
		rewardSubmissions, err := rs.handleRewardSubmissionCreatedEvent(log)
		if err != nil {
			return nil, err
		}

		for _, rewardSubmission := range rewardSubmissions.Submissions {
			slotId := NewSlotId(rewardSubmission.RewardHash, rewardSubmission.Strategy)

			_, ok := rs.stateAccumulator[log.BlockNumber][slotId]
			if ok {
				err := xerrors.Errorf("Duplicate distribution root submitted for slot %s at block %d", slotId, log.BlockNumber)
				rs.logger.Sugar().Errorw("Duplicate distribution root submitted", zap.Error(err))
				return nil, err
			}

			rs.stateAccumulator[log.BlockNumber][slotId] = rewardSubmission
		}

		return rewardSubmissions, nil
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

func (rs *RewardSubmissionsModel) getContractAddressesForEnvironment() map[string][]string {
	contracts := rs.globalConfig.GetContractsMapForEnvAndNetwork()
	return map[string][]string{
		contracts.RewardsCoordinator: []string{
			"RangePaymentForAllCreated",
			"RewardsSubmissionForAllCreated",
			"RangePaymentCreated",
			"AVSRewardsSubmissionCreated",
		},
	}
}

func (rs *RewardSubmissionsModel) IsInterestingLog(log *storage.TransactionLog) bool {
	addresses := rs.getContractAddressesForEnvironment()
	return rs.BaseEigenState.IsInterestingLog(addresses, log)
}

func (rs *RewardSubmissionsModel) InitBlockProcessing(blockNumber uint64) error {
	rs.stateAccumulator[blockNumber] = make(map[SlotId]*RewardSubmission)
	return nil
}

func (rs *RewardSubmissionsModel) HandleStateChange(log *storage.TransactionLog) (interface{}, error) {
	stateChanges, sortedBlockNumbers := rs.GetStateTransitions()

	for _, blockNumber := range sortedBlockNumbers {
		if log.BlockNumber >= blockNumber {
			rs.logger.Sugar().Debugw("Handling state change", zap.Uint64("blockNumber", blockNumber))

			change, err := stateChanges[blockNumber](log)
			if err != nil {
				return nil, err
			}
			if change == nil {
				return nil, nil
			}
			return change, nil
		}
	}
	return nil, nil
}

func (rs *RewardSubmissionsModel) clonePreviousBlocksToNewBlock(blockNumber uint64) error {
	query := `
		insert into reward_submissions(avs, reward_hash, token, amount, strategy, strategy_index, multiplier, start_timestamp, end_timestamp, duration, is_for_all, block_number)
			select
				avs,
				reward_hash,
				token,
				amount,
				strategy,
				strategy_index,
				multiplier,
				start_timestamp,
				end_timestamp,
				duration,
				is_for_all,
				@currentBlock as block_number
			from reward_submissions
			where block_number = @previousBlock
	`
	res := rs.Db.Exec(query,
		sql.Named("currentBlock", blockNumber),
		sql.Named("previousBlock", blockNumber-1),
	)

	if res.Error != nil {
		rs.logger.Sugar().Errorw("Failed to clone previous block state to new block", zap.Error(res.Error))
		return res.Error
	}
	return nil
}

// prepareState prepares the state for commit by adding the new state to the existing state
func (rs *RewardSubmissionsModel) prepareState(blockNumber uint64) ([]*RewardSubmissionDiff, []*RewardSubmissionDiff, error) {
	accumulatedState, ok := rs.stateAccumulator[blockNumber]
	if !ok {
		err := xerrors.Errorf("No accumulated state found for block %d", blockNumber)
		rs.logger.Sugar().Errorw(err.Error(), zap.Error(err), zap.Uint64("blockNumber", blockNumber))
		return nil, nil, err
	}

	currentBlock := &storage.Block{}
	err := rs.Db.Where("number = ?", blockNumber).First(currentBlock).Error
	if err != nil {
		rs.logger.Sugar().Errorw("Failed to fetch block", zap.Error(err), zap.Uint64("blockNumber", blockNumber))
		return nil, nil, err
	}

	inserts := make([]*RewardSubmissionDiff, 0)
	for _, change := range accumulatedState {
		if change == nil {
			continue
		}

		inserts = append(inserts, &RewardSubmissionDiff{
			RewardSubmission: change,
			IsNew:            true,
		})
	}

	// find all the records that are no longer active
	noLongerActiveSubmissions := make([]*RewardSubmission, 0)
	query := `
		select
			*
		from reward_submissions
		where
			block_number = @previousBlock
			and end_timestamp <= @blockTime
	`
	res := rs.Db.
		Model(&RewardSubmission{}).
		Raw(query,
			sql.Named("previousBlock", blockNumber-1),
			sql.Named("blockTime", currentBlock.BlockTime.Unix()),
		).
		Find(&noLongerActiveSubmissions)

	if res.Error != nil {
		rs.logger.Sugar().Errorw("Failed to fetch no longer active submissions", zap.Error(res.Error))
		return nil, nil, res.Error
	}

	deletes := make([]*RewardSubmissionDiff, 0)
	for _, submission := range noLongerActiveSubmissions {
		deletes = append(deletes, &RewardSubmissionDiff{
			RewardSubmission: submission,
			IsNoLongerActive: true,
		})
	}
	return inserts, deletes, nil
}

// CommitFinalState commits the final state for the given block number
func (rs *RewardSubmissionsModel) CommitFinalState(blockNumber uint64) error {
	err := rs.clonePreviousBlocksToNewBlock(blockNumber)
	if err != nil {
		return err
	}

	recordsToInsert, recordsToDelete, err := rs.prepareState(blockNumber)
	if err != nil {
		return err
	}

	for _, record := range recordsToDelete {
		res := rs.Db.Delete(&RewardSubmission{}, "reward_hash = ? and strategy = ? and block_number = ?", record.RewardSubmission.RewardHash, record.RewardSubmission.Strategy, blockNumber)
		if res.Error != nil {
			rs.logger.Sugar().Errorw("Failed to delete record",
				zap.Error(res.Error),
				zap.String("rewardHash", record.RewardSubmission.RewardHash),
				zap.String("strategy", record.RewardSubmission.Strategy),
				zap.Uint64("blockNumber", blockNumber),
			)
			return res.Error
		}
	}
	if len(recordsToInsert) > 0 {
		// records := make([]RewardSubmission, 0)
		for _, record := range recordsToInsert {
			res := rs.Db.Model(&RewardSubmission{}).Clauses(clause.Returning{}).Create(&record.RewardSubmission)
			if res.Error != nil {
				rs.logger.Sugar().Errorw("Failed to insert records", zap.Error(res.Error))
				fmt.Printf("\n\n%+v\n\n", record.RewardSubmission)
				return res.Error
			}
		}
	}
	return nil
}

func (rs *RewardSubmissionsModel) ClearAccumulatedState(blockNumber uint64) error {
	delete(rs.stateAccumulator, blockNumber)
	return nil
}

// GenerateStateRoot generates the state root for the given block number using the results of the state changes
func (rs *RewardSubmissionsModel) GenerateStateRoot(blockNumber uint64) (types.StateRoot, error) {
	inserts, deletes, err := rs.prepareState(blockNumber)
	if err != nil {
		return "", err
	}

	combinedResults := make([]*RewardSubmissionDiff, 0)
	for _, record := range inserts {
		combinedResults = append(combinedResults, record)
	}
	for _, record := range deletes {
		combinedResults = append(combinedResults, record)
	}

	fullTree, err := rs.merkelizeState(blockNumber, combinedResults)
	if err != nil {
		return "", err
	}
	return types.StateRoot(utils.ConvertBytesToString(fullTree.Root())), nil
}

func (rs *RewardSubmissionsModel) sortRewardSubmissionsForMerkelization(submissions []*RewardSubmissionDiff) []*RewardSubmissionDiff {
	mappedByAvs := make(map[string][]*RewardSubmissionDiff)
	for _, submission := range submissions {
		if _, ok := mappedByAvs[submission.RewardSubmission.Avs]; !ok {
			mappedByAvs[submission.RewardSubmission.Avs] = make([]*RewardSubmissionDiff, 0)
		}
		mappedByAvs[submission.RewardSubmission.Avs] = append(mappedByAvs[submission.RewardSubmission.Avs], submission)
	}

	for _, sub := range mappedByAvs {
		slices.SortFunc(sub, func(i, j *RewardSubmissionDiff) int {
			iSlotId := NewSlotId(i.RewardSubmission.RewardHash, i.RewardSubmission.Strategy)
			jSlotId := NewSlotId(j.RewardSubmission.RewardHash, j.RewardSubmission.Strategy)

			return strings.Compare(string(iSlotId), string(jSlotId))
		})
	}

	avsAddresses := make([]string, 0)
	for key, _ := range mappedByAvs {
		avsAddresses = append(avsAddresses, key)
	}

	sort.Slice(avsAddresses, func(i, j int) bool {
		return avsAddresses[i] < avsAddresses[j]
	})

	sorted := make([]*RewardSubmissionDiff, 0)
	for _, avs := range avsAddresses {
		sorted = append(sorted, mappedByAvs[avs]...)
	}
	return sorted
}

func (rs *RewardSubmissionsModel) merkelizeState(blockNumber uint64, rewardSubmissions []*RewardSubmissionDiff) (*merkletree.MerkleTree, error) {
	// Avs -> slot_id -> string (added/removed)
	om := orderedmap.New[string, *orderedmap.OrderedMap[SlotId, string]]()

	rewardSubmissions = rs.sortRewardSubmissionsForMerkelization(rewardSubmissions)

	for _, result := range rewardSubmissions {
		existingAvs, found := om.Get(result.RewardSubmission.Avs)
		if !found {
			existingAvs = orderedmap.New[SlotId, string]()
			om.Set(result.RewardSubmission.Avs, existingAvs)

			prev := om.GetPair(result.RewardSubmission.Avs).Prev()
			if prev != nil && strings.Compare(prev.Key, result.RewardSubmission.Avs) >= 0 {
				om.Delete(result.RewardSubmission.Avs)
				return nil, fmt.Errorf("avs not in order")
			}
		}
		slotId := NewSlotId(result.RewardSubmission.RewardHash, result.RewardSubmission.Strategy)
		var state string
		if result.IsNew {
			state = "added"
		} else if result.IsNoLongerActive {
			state = "removed"
		} else {
			return nil, fmt.Errorf("invalid state change")
		}
		existingAvs.Set(slotId, state)

		prev := existingAvs.GetPair(slotId).Prev()
		if prev != nil && strings.Compare(string(prev.Key), string(slotId)) >= 0 {
			existingAvs.Delete(slotId)
			return nil, fmt.Errorf("operator not in order")
		}
	}

	avsLeaves := rs.InitializeMerkleTreeBaseStateWithBlock(blockNumber)

	for avs := om.Oldest(); avs != nil; avs = avs.Next() {
		submissionLeafs := make([][]byte, 0)
		for submission := avs.Value.Oldest(); submission != nil; submission = submission.Next() {
			slotId := submission.Key
			state := submission.Value
			submissionLeafs = append(submissionLeafs, encodeSubmissionLeaf(slotId, state))
		}

		avsTree, err := merkletree.NewTree(
			merkletree.WithData(submissionLeafs),
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

func encodeSubmissionLeaf(slotId SlotId, state string) []byte {
	return []byte(fmt.Sprintf("%s:%s", slotId, state))
}

func encodeAvsLeaf(avs string, avsSubmissionRoot []byte) []byte {
	return append([]byte(avs), avsSubmissionRoot[:]...)
}

func (rs *RewardSubmissionsModel) DeleteState(startBlockNumber uint64, endBlockNumber uint64) error {
	return rs.BaseEigenState.DeleteState("registered_avs_operators", startBlockNumber, endBlockNumber, rs.Db)
}
