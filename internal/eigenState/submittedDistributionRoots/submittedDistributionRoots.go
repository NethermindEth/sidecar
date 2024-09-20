package submittedDistributionRoots

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/Layr-Labs/go-sidecar/internal/config"
	"github.com/Layr-Labs/go-sidecar/internal/eigenState/base"
	"github.com/Layr-Labs/go-sidecar/internal/eigenState/stateManager"
	"github.com/Layr-Labs/go-sidecar/internal/eigenState/types"
	"github.com/Layr-Labs/go-sidecar/internal/storage"
	"github.com/Layr-Labs/go-sidecar/internal/utils"
	"go.uber.org/zap"
	"golang.org/x/xerrors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type SubmittedDistributionRoots struct {
	Root                      string
	BlockNumber               uint64
	RootIndex                 uint64
	RewardsCalculationEnd     string
	RewardsCalculationEndUnit string
	ActivatedAt               string
	ActivatedAtUnit           string
	CreatedAtBlockNumber      uint64
}

func NewSlotID(root string, rootIndex uint64) types.SlotID {
	return types.SlotID(fmt.Sprintf("%s_%d", root, rootIndex))
}

type SubmittedDistributionRootsModel struct {
	base.BaseEigenState
	StateTransitions types.StateTransitions[SubmittedDistributionRoots]
	DB               *gorm.DB
	logger           *zap.Logger
	globalConfig     *config.Config

	// Accumulates state changes for SlotIds, grouped by block number
	stateAccumulator map[uint64]map[types.SlotID]*SubmittedDistributionRoots
}

func NewSubmittedDistributionRootsModel(
	esm *stateManager.EigenStateManager,
	grm *gorm.DB,
	logger *zap.Logger,
	globalConfig *config.Config,
) (*SubmittedDistributionRootsModel, error) {
	model := &SubmittedDistributionRootsModel{
		BaseEigenState: base.BaseEigenState{
			Logger: logger,
		},
		DB:               grm,
		logger:           logger,
		globalConfig:     globalConfig,
		stateAccumulator: make(map[uint64]map[types.SlotID]*SubmittedDistributionRoots),
	}

	esm.RegisterState(model, 4)
	return model, nil
}

func (sdr *SubmittedDistributionRootsModel) GetModelName() string {
	return "SubmittedDistributionRootsModel"
}

type distributionRootSubmittedOutput struct {
	ActivatedAt uint64 `json:"activatedAt"`
}

func parseLogOutputForDistributionRootSubmitted(outputDataStr string) (*distributionRootSubmittedOutput, error) {
	outputData := &distributionRootSubmittedOutput{}
	decoder := json.NewDecoder(strings.NewReader(outputDataStr))
	decoder.UseNumber()

	err := decoder.Decode(&outputData)
	if err != nil {
		return nil, err
	}
	return outputData, err
}

func (sdr *SubmittedDistributionRootsModel) GetStateTransitions() (types.StateTransitions[SubmittedDistributionRoots], []uint64) {
	stateChanges := make(types.StateTransitions[SubmittedDistributionRoots])

	stateChanges[0] = func(log *storage.TransactionLog) (*SubmittedDistributionRoots, error) {
		arguments, err := sdr.ParseLogArguments(log)
		if err != nil {
			return nil, err
		}
		outputData, err := parseLogOutputForDistributionRootSubmitted(log.OutputData)
		if err != nil {
			return nil, err
		}

		// Sanity check to make sure we've got an initialized accumulator map for the block
		if _, ok := sdr.stateAccumulator[log.BlockNumber]; !ok {
			return nil, xerrors.Errorf("No state accumulator found for block %d", log.BlockNumber)
		}

		var rootIndex uint64

		t := reflect.TypeOf(arguments[0].Value)
		switch t.Kind() {
		case reflect.String:
			if arguments[0].Value.(string) == "0x0000000000000000000000000000000000000000" {
				rootIndex = 0
				break
			}
			withoutPrefix := strings.TrimPrefix(arguments[2].Value.(string), "0x")
			rootIndex, err = strconv.ParseUint(withoutPrefix, 16, 32)
			if err != nil {
				return nil, xerrors.Errorf("Failed to decode rootIndex: %v", err)
			}
		case reflect.Float64:
			rootIndex = uint64(arguments[0].Value.(float64))
		default:
			return nil, xerrors.Errorf("Invalid type for rootIndex: %s", t.Kind())
		}

		root := arguments[1].Value.(string)

		var rewardsCalculationEnd string
		calcualtionEndType := reflect.TypeOf(arguments[2].Value)
		switch calcualtionEndType.Kind() {
		case reflect.String:
			withoutPrefix := strings.TrimPrefix(arguments[2].Value.(string), "0x")
			decoded, err := strconv.ParseUint(withoutPrefix, 16, 32)
			if err != nil {
				return nil, xerrors.Errorf("Failed to decode rewardsCalculationEnd: %v", err)
			}
			rewardsCalculationEnd = fmt.Sprintf("%d", decoded)
		case reflect.Float64:
			rewardsCalculationEnd = fmt.Sprintf("%d", uint64(arguments[2].Value.(float64)))
		default:
			return nil, xerrors.Errorf("Invalid type for rewardsCalculationEnd: %s", calcualtionEndType.Kind())
		}

		activatedAt := outputData.ActivatedAt

		slotId := NewSlotID(root, rootIndex)
		_, ok := sdr.stateAccumulator[log.BlockNumber][slotId]
		if ok {
			err := xerrors.Errorf("Duplicate distribution root submitted for slot %s at block %d", slotId, log.BlockNumber)
			sdr.logger.Sugar().Errorw("Duplicate distribution root submitted", zap.Error(err))
			return nil, err
		}

		record := &SubmittedDistributionRoots{
			Root:                      root,
			BlockNumber:               log.BlockNumber,
			RootIndex:                 rootIndex,
			RewardsCalculationEnd:     rewardsCalculationEnd,
			RewardsCalculationEndUnit: "snapshot",
			ActivatedAt:               fmt.Sprintf("%d", activatedAt),
			ActivatedAtUnit:           "timestamp",
			CreatedAtBlockNumber:      log.BlockNumber,
		}
		sdr.stateAccumulator[log.BlockNumber][slotId] = record

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

func (sdr *SubmittedDistributionRootsModel) getContractAddressesForEnvironment() map[string][]string {
	contracts := sdr.globalConfig.GetContractsMapForChain()
	return map[string][]string{
		contracts.RewardsCoordinator: {
			"DistributionRootSubmitted",
		},
	}
}

func (sdr *SubmittedDistributionRootsModel) IsInterestingLog(log *storage.TransactionLog) bool {
	addresses := sdr.getContractAddressesForEnvironment()
	return sdr.BaseEigenState.IsInterestingLog(addresses, log)
}

func (sdr *SubmittedDistributionRootsModel) InitBlockProcessing(blockNumber uint64) error {
	sdr.stateAccumulator[blockNumber] = make(map[types.SlotID]*SubmittedDistributionRoots)
	return nil
}

func (sdr *SubmittedDistributionRootsModel) HandleStateChange(log *storage.TransactionLog) (interface{}, error) {
	stateChanges, sortedBlockNumbers := sdr.GetStateTransitions()

	for _, blockNumber := range sortedBlockNumbers {
		if log.BlockNumber >= blockNumber {
			sdr.logger.Sugar().Debugw("Handling state change", zap.Uint64("blockNumber", blockNumber))

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

func (sdr *SubmittedDistributionRootsModel) clonePreviousBlocksToNewBlock(blockNumber uint64) error {
	query := `
		insert into submitted_distribution_roots (root, root_index, rewards_calculation_end, rewards_calculation_end_unit, activated_at, activated_at_unit, created_at_block_number, block_number)
			select
				root,
				root_index,
				rewards_calculation_end,
				rewards_calculation_end_unit,
				activated_at,
				activated_at_unit,
				created_at_block_number,
				@currentBlock as block_number
			from submitted_distribution_roots
			where block_number = @previousBlock
	`
	res := sdr.DB.Exec(query,
		sql.Named("currentBlock", blockNumber),
		sql.Named("previousBlock", blockNumber-1),
	)

	if res.Error != nil {
		sdr.logger.Sugar().Errorw("Failed to clone previous block state to new block", zap.Error(res.Error))
		return res.Error
	}
	return nil
}

// prepareState prepares the state for commit by adding the new state to the existing state.
func (sdr *SubmittedDistributionRootsModel) prepareState(blockNumber uint64) ([]SubmittedDistributionRoots, error) {
	preparedState := make([]SubmittedDistributionRoots, 0)

	accumulatedState, ok := sdr.stateAccumulator[blockNumber]
	if !ok {
		err := xerrors.Errorf("No accumulated state found for block %d", blockNumber)
		sdr.logger.Sugar().Errorw(err.Error(), zap.Error(err), zap.Uint64("blockNumber", blockNumber))
		return nil, err
	}

	slotIds := make([]types.SlotID, 0)
	for slotId := range accumulatedState {
		slotIds = append(slotIds, slotId)
	}

	// Find only the records from the previous block, that are modified in this block
	query := `
		select
			root,
			root_index,
			rewards_calculation_end,
			rewards_calculation_end_unit,
			activated_at,
			activated_at_unit,
			block_number
		from submitted_distribution_roots
		where
			block_number < @currentBlock
			and concat(root, '_', root_index) in @slotIds
	`
	existingRecords := make([]SubmittedDistributionRoots, 0)
	res := sdr.DB.Model(&SubmittedDistributionRoots{}).
		Raw(query,
			sql.Named("currentBlock", blockNumber),
			sql.Named("slotIds", slotIds),
		).
		Scan(&existingRecords)

	if res.Error != nil {
		sdr.logger.Sugar().Errorw("Failed to fetch submitted_distribution_roots", zap.Error(res.Error))
		return nil, res.Error
	}

	if len(existingRecords) > 0 {
		sdr.logger.Sugar().Debugw("Found slotIds that already exist", zap.Int("count", len(existingRecords)))
		return nil, xerrors.Errorf("Found slotIds that already exist")
	}

	for _, newState := range accumulatedState {
		prepared := SubmittedDistributionRoots{
			Root:                      newState.Root,
			BlockNumber:               blockNumber,
			RootIndex:                 newState.RootIndex,
			RewardsCalculationEnd:     newState.RewardsCalculationEnd,
			RewardsCalculationEndUnit: newState.RewardsCalculationEndUnit,
			ActivatedAt:               newState.ActivatedAt,
			ActivatedAtUnit:           newState.ActivatedAtUnit,
			CreatedAtBlockNumber:      newState.CreatedAtBlockNumber,
		}

		preparedState = append(preparedState, prepared)
	}
	return preparedState, nil
}

func (sdr *SubmittedDistributionRootsModel) CommitFinalState(blockNumber uint64) error {
	err := sdr.clonePreviousBlocksToNewBlock(blockNumber)
	if err != nil {
		return err
	}

	records, err := sdr.prepareState(blockNumber)
	if err != nil {
		return err
	}

	if len(records) > 0 {
		res := sdr.DB.Model(&SubmittedDistributionRoots{}).Clauses(clause.Returning{}).Create(&records)
		if res.Error != nil {
			sdr.logger.Sugar().Errorw("Failed to create new submitted_distribution_roots records", zap.Error(res.Error))
			return res.Error
		}
	}

	return nil
}

func (sdr *SubmittedDistributionRootsModel) ClearAccumulatedState(blockNumber uint64) error {
	delete(sdr.stateAccumulator, blockNumber)
	return nil
}

func (sdr *SubmittedDistributionRootsModel) sortValuesForMerkleTree(inputs []SubmittedDistributionRoots) []*base.MerkleTreeInput {
	slices.SortFunc(inputs, func(i, j SubmittedDistributionRoots) int {
		return int(i.RootIndex - j.RootIndex)
	})

	values := make([]*base.MerkleTreeInput, 0)
	for _, input := range inputs {
		values = append(values, &base.MerkleTreeInput{
			SlotID: NewSlotID(input.Root, input.RootIndex),
			Value:  []byte(input.Root),
		})
	}
	return values
}

func (sdr *SubmittedDistributionRootsModel) GenerateStateRoot(blockNumber uint64) (types.StateRoot, error) {
	diffs, err := sdr.prepareState(blockNumber)
	if err != nil {
		return "", err
	}

	sortedInputs := sdr.sortValuesForMerkleTree(diffs)

	fullTree, err := sdr.MerkleizeState(blockNumber, sortedInputs)
	if err != nil {
		return "", err
	}
	return types.StateRoot(utils.ConvertBytesToString(fullTree.Root())), nil
}

func (sdr *SubmittedDistributionRootsModel) DeleteState(startBlockNumber uint64, endBlockNumber uint64) error {
	return sdr.BaseEigenState.DeleteState("submitted_distribution_roots", startBlockNumber, endBlockNumber, sdr.DB)
}
