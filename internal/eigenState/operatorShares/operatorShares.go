package operatorShares

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/Layr-Labs/sidecar/internal/config"
	"github.com/Layr-Labs/sidecar/internal/eigenState"
	"github.com/Layr-Labs/sidecar/internal/parser"
	"github.com/Layr-Labs/sidecar/internal/storage"
	"github.com/Layr-Labs/sidecar/internal/utils"
	"github.com/wealdtech/go-merkletree/v2"
	"github.com/wealdtech/go-merkletree/v2/keccak256"
	orderedmap "github.com/wk8/go-ordered-map/v2"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"math/big"
	"slices"
	"sort"
	"strings"
	"time"
)

// Changes table
type OperatorShareChange struct {
	Id               uint64 `gorm:"type:serial"`
	Operator         string
	Strategy         string
	Shares           string `gorm:"type:numeric"`
	TransactionHash  string
	TransactionIndex uint64
	LogIndex         uint64
	BlockNumber      uint64
	CreatedAt        time.Time
}

// Block table
type OperatorShares struct {
	Operator    string
	Strategy    string
	Shares      string `gorm:"type:numeric"`
	BlockNumber uint64
	CreatedAt   time.Time
}

// Implements IEigenStateModel
type OperatorSharesModel struct {
	eigenState.BaseEigenState
	StateTransitions eigenState.StateTransitions[OperatorShareChange]
	Db               *gorm.DB
	Network          config.Network
	Environment      config.Environment
	logger           *zap.Logger
	globalConfig     *config.Config
}

func NewOperatorSharesModel(
	esm *eigenState.EigenStateManager,
	grm *gorm.DB,
	Network config.Network,
	Environment config.Environment,
	logger *zap.Logger,
	globalConfig *config.Config,
) (*OperatorSharesModel, error) {
	model := &OperatorSharesModel{
		BaseEigenState: eigenState.BaseEigenState{},
		Db:             grm,
		Network:        Network,
		Environment:    Environment,
		logger:         logger,
		globalConfig:   globalConfig,
	}

	esm.RegisterState(model)
	return model, nil
}

func (osm *OperatorSharesModel) GetStateTransitions() (eigenState.StateTransitions[OperatorShareChange], []uint64) {
	stateChanges := make(eigenState.StateTransitions[OperatorShareChange])

	stateChanges[0] = func(log *storage.TransactionLog) (*OperatorShareChange, error) {
		arguments := make([]parser.Argument, 0)
		err := json.Unmarshal([]byte(log.Arguments), &arguments)
		if err != nil {
			osm.logger.Sugar().Errorw("Failed to unmarshal arguments",
				zap.Error(err),
				zap.String("transactionHash", log.TransactionHash),
				zap.Uint64("transactionIndex", log.TransactionIndex),
			)
			return nil, err
		}
		outputData := make(map[string]interface{})
		err = json.Unmarshal([]byte(log.OutputData), &outputData)
		if err != nil {
			osm.logger.Sugar().Errorw("Failed to unmarshal outputData",
				zap.Error(err),
				zap.String("transactionHash", log.TransactionHash),
				zap.Uint64("transactionIndex", log.TransactionIndex),
			)
			return nil, err
		}
		shares := big.Int{}
		sharesInt, _ := shares.SetString(outputData["shares"].(string), 10)

		if log.EventName == "OperatorSharesDecreased" {
			sharesInt.Mul(sharesInt, big.NewInt(-1))
		}

		change := &OperatorShareChange{
			Operator:         arguments[0].Value.(string),
			Strategy:         outputData["strategy"].(string),
			Shares:           sharesInt.String(),
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
	logAddress := strings.ToLower(log.Address)
	if eventNames, ok := addresses[logAddress]; ok {
		if slices.Contains(eventNames, log.EventName) {
			return true
		}
	}
	return false
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

			if change != nil {
				wroteChange, err := osm.writeStateChange(change)
				if err != nil {
					return wroteChange, err
				}
				return wroteChange, nil
			}
		}
	}
	return nil, nil
}

func (osm *OperatorSharesModel) writeStateChange(change *OperatorShareChange) (interface{}, error) {
	osm.logger.Sugar().Debugw("Writing state change", zap.Any("change", change))
	res := osm.Db.Model(&OperatorShareChange{}).Clauses(clause.Returning{}).Create(change)
	if res.Error != nil {
		osm.logger.Error("Failed to insert into avs_operator_changes", zap.Error(res.Error))
		return change, res.Error
	}
	return change, nil
}

func (osm *OperatorSharesModel) WriteFinalState(blockNumber uint64) error {
	query := `
		with new_sum as (
			select
				operator,
				strategy,
				sum(shares) as shares
			from
				operator_share_changes
			where
				block_number = @currentBlock
			group by 1, 2
		),
		previous_values as (
			select
				operator,
				strategy,
				shares
			from operator_shares
			where block_number = @previousBlock
		),
		unioned_values as (
			(select operator, strategy, shares from previous_values)
			union
			(select operator, strategy, shares from new_sum)
		),
		final_values as (
			select
				operator,
				strategy,
				sum(shares) as shares
			from unioned_values
			group by 1, 2
		)
		insert into operator_shares (operator, strategy, shares, block_number)
			select operator, strategy, shares, @currentBlock as block_number from final_values
	`

	res := osm.Db.Exec(query,
		sql.Named("currentBlock", blockNumber),
		sql.Named("previousBlock", blockNumber-1),
	)
	if res.Error != nil {
		osm.logger.Sugar().Errorw("Failed to insert into operator_shares", zap.Error(res.Error))
		return res.Error
	}
	return nil
}

func (osm *OperatorSharesModel) getDifferencesInStates(currentBlock uint64) ([]OperatorShares, error) {
	query := `
		with new_states as (
			select
				concat(operator, '_', strategy) as slot_id,
				operator,
				strategy,
				shares
			from operator_shares
			where block_number = @currentBlock
		),
		previous_states as (
			select
				concat(operator, '_', strategy) as slot_id,
				operator,
				strategy,
				shares
			from operator_shares
			where block_number = @previousBlock
		),
		diffs as (
			select slot_id, operator, strategy, shares from new_states
			except
			select slot_id, operator, strategy, shares from previous_states
		)
		select operator, strategy, shares from diffs
		order by strategy asc, operator asc
	`

	diffs := make([]OperatorShares, 0)
	res := osm.Db.
		Raw(query,
			sql.Named("currentBlock", currentBlock),
			sql.Named("previousBlock", currentBlock-1),
		).
		Scan(&diffs)
	if res.Error != nil {
		osm.logger.Sugar().Errorw("Failed to fetch operator_shares", zap.Error(res.Error))
		return nil, res.Error
	}
	return diffs, nil
}

func (osm *OperatorSharesModel) GenerateStateRoot(blockNumber uint64) (eigenState.StateRoot, error) {
	diffs, err := osm.getDifferencesInStates(blockNumber)
	if err != nil {
		return "", err
	}

	fullTree, err := osm.merkelizeState(diffs)
	if err != nil {
		return "", err
	}
	return eigenState.StateRoot(utils.ConvertBytesToString(fullTree.Root())), nil
}

func (osm *OperatorSharesModel) merkelizeState(diffs []OperatorShares) (*merkletree.MerkleTree, error) {
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
		existingStrategy.Set(diff.Operator, diff.Shares)

		prev := existingStrategy.GetPair(diff.Operator).Prev()
		if prev != nil && strings.Compare(prev.Key, diff.Operator) >= 0 {
			existingStrategy.Delete(diff.Operator)
			return nil, fmt.Errorf("operator not in order")
		}
	}

	leaves := make([][]byte, 0)
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
