package rewardsClaimed

import (
	"encoding/json"
	"fmt"
	"github.com/Layr-Labs/sidecar/internal/config"
	"github.com/Layr-Labs/sidecar/pkg/metaState/baseModel"
	"github.com/Layr-Labs/sidecar/pkg/metaState/metaStateManager"
	"github.com/Layr-Labs/sidecar/pkg/metaState/types"
	"github.com/Layr-Labs/sidecar/pkg/storage"
	"github.com/Layr-Labs/sidecar/pkg/utils"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"strings"
)

type RewardsClaimedModel struct {
	db           *gorm.DB
	logger       *zap.Logger
	globalConfig *config.Config

	accumulatedState map[uint64][]*types.RewardsClaimed
}

func NewRewardsClaimedModel(
	db *gorm.DB,
	logger *zap.Logger,
	globalConfig *config.Config,
	msm *metaStateManager.MetaStateManager,
) (*RewardsClaimedModel, error) {
	model := &RewardsClaimedModel{
		db:               db,
		logger:           logger,
		globalConfig:     globalConfig,
		accumulatedState: make(map[uint64][]*types.RewardsClaimed),
	}
	msm.RegisterMetaStateModel(model)
	return model, nil
}

const RewardsClaimedModelTableName = "rewards_claimed"

func (rcm *RewardsClaimedModel) TableName() string {
	return RewardsClaimedModelTableName
}

func (rcm *RewardsClaimedModel) SetupStateForBlock(blockNumber uint64) error {
	rcm.accumulatedState[blockNumber] = make([]*types.RewardsClaimed, 0)
	return nil
}

func (rcm *RewardsClaimedModel) CleanupStateForBlock(blockNumber uint64) error {
	delete(rcm.accumulatedState, blockNumber)
	return nil
}

func (rcm *RewardsClaimedModel) getContractAddressesForEnvironment() map[string][]string {
	contracts := rcm.globalConfig.GetContractsMapForChain()
	return map[string][]string{
		contracts.RewardsCoordinator: {
			"RewardsClaimed",
		},
	}
}

func (rcm *RewardsClaimedModel) IsInterestingLog(log *storage.TransactionLog) bool {
	contracts := rcm.getContractAddressesForEnvironment()
	return baseModel.IsInterestingLog(contracts, log)
}

type LogOutput struct {
	Root          []byte      `json:"root"`
	Token         string      `json:"token"`
	ClaimedAmount json.Number `json:"claimedAmount"`
}

func (rcm *RewardsClaimedModel) HandleTransactionLog(log *storage.TransactionLog) (interface{}, error) {
	arguments, err := baseModel.ParseLogArguments(log, rcm.logger)
	if err != nil {
		return nil, err
	}
	outputData, err := baseModel.ParseLogOutput[LogOutput](log, rcm.logger)
	if err != nil {
		return nil, err
	}

	rootString := utils.ConvertBytesToString(outputData.Root)

	var recipient string
	if arguments[3].Value != nil {
		recipient = arguments[3].Value.(string)
	} else {
		recipient = ""
	}

	claimed := &types.RewardsClaimed{
		Root:            rootString,
		Earner:          strings.ToLower(arguments[1].Value.(string)),
		Claimer:         strings.ToLower(arguments[2].Value.(string)),
		Recipient:       recipient,
		Token:           outputData.Token,
		ClaimedAmount:   outputData.ClaimedAmount.String(),
		BlockNumber:     log.BlockNumber,
		TransactionHash: log.TransactionHash,
		LogIndex:        log.LogIndex,
	}

	rcm.accumulatedState[log.BlockNumber] = append(rcm.accumulatedState[log.BlockNumber], claimed)
	return claimed, nil
}

func (rcm *RewardsClaimedModel) CommitFinalState(blockNumber uint64) error {
	rowsToInsert, ok := rcm.accumulatedState[blockNumber]
	if !ok {
		return fmt.Errorf("block number not initialized in accumulatedState %d", blockNumber)
	}

	if len(rowsToInsert) == 0 {
		rcm.logger.Sugar().Debugf("No rewards claimed to insert for block %d", blockNumber)
		return nil
	}

	res := rcm.db.Model(&types.RewardsClaimed{}).Clauses(clause.Returning{}).Create(&rowsToInsert)
	if res.Error != nil {
		rcm.logger.Sugar().Errorw("Failed to insert rewards claimed records", zap.Error(res.Error))
		return res.Error
	}
	return nil
}

func (rcm *RewardsClaimedModel) DeleteState(startBlockNumber uint64, endBlockNumber uint64) error {
	return baseModel.DeleteState(rcm.TableName(), startBlockNumber, endBlockNumber, rcm.db, rcm.logger)
}
