package multicallContractCaller

import (
	"context"
	"errors"
	"fmt"
	"github.com/Layr-Labs/eigenlayer-contracts/pkg/bindings/IRewardsCoordinator"
	"github.com/Layr-Labs/go-sidecar/internal/multicall"
	"github.com/Layr-Labs/go-sidecar/pkg/clients/ethereum"
	"github.com/Layr-Labs/go-sidecar/pkg/contractCaller"
	"github.com/Layr-Labs/go-sidecar/pkg/utils"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"go.uber.org/zap"
	"math/big"
	"strings"
)

type MulticallContractCaller struct {
	EthereumClient *ethereum.Client
	Logger         *zap.Logger
}

func NewMulticallContractCaller(ec *ethereum.Client, l *zap.Logger) *MulticallContractCaller {
	return &MulticallContractCaller{
		EthereumClient: ec,
		Logger:         l,
	}
}

func getOperatorRestakedStrategies(ctx context.Context, avs string, operator string, blockNumber uint64, client *ethereum.Client, l *zap.Logger) ([]common.Address, error) {
	a, err := abi.JSON(strings.NewReader(contractCaller.AvsServiceManagerAbi))
	if err != nil {
		l.Sugar().Errorw("getOperatorRestakedStrategies - failed to parse abi", zap.Error(err))
		return nil, err
	}

	callerClient, err := client.GetEthereumContractCaller()
	if err != nil {
		l.Sugar().Errorw("getOperatorRestakedStrategies - failed to get contract caller", zap.Error(err))
		return nil, err
	}

	contract := bind.NewBoundContract(common.HexToAddress(avs), a, callerClient, nil, nil)

	bigBlockNumber := big.NewInt(int64(blockNumber))

	results := make([]interface{}, 0)

	err = contract.Call(&bind.CallOpts{BlockNumber: bigBlockNumber, Context: ctx}, &results, "getOperatorRestakedStrategies", common.HexToAddress(operator))
	if err != nil {
		l.Sugar().Errorw("getOperatorRestakedStrategies - failed to call contract method", zap.Error(err))
		return nil, err
	}

	return results[0].([]common.Address), nil
}

func (cc *MulticallContractCaller) GetOperatorRestakedStrategies(ctx context.Context, avs string, operator string, blockNumber uint64) ([]common.Address, error) {
	return getOperatorRestakedStrategies(ctx, avs, operator, blockNumber, cc.EthereumClient, cc.Logger)
}

func (cc *MulticallContractCaller) GetAllOperatorRestakedStrategies(
	ctx context.Context,
	operatorRestakedStrategies []*contractCaller.OperatorRestakedStrategy,
	blockNumber uint64,
) ([]*contractCaller.OperatorRestakedStrategy, error) {
	a, err := abi.JSON(strings.NewReader(contractCaller.AvsServiceManagerAbi))
	if err != nil {
		cc.Logger.Sugar().Errorw("getOperatorRestakedStrategies - failed to parse abi", zap.Error(err))
		return nil, err
	}

	type MulticallAndError struct {
		Multicall *multicall.MultiCallMetaData[[]common.Address]
		Error     error
	}

	requests := utils.Map(operatorRestakedStrategies, func(ors *contractCaller.OperatorRestakedStrategy, index uint64) *MulticallAndError {
		mc, err := multicall.Describe[[]common.Address](common.HexToAddress(ors.Avs), a, "getOperatorRestakedStrategies", common.HexToAddress(ors.Operator))
		return &MulticallAndError{
			Multicall: mc,
			Error:     err,
		}
	})

	errs := []error{}
	for _, mc := range requests {
		if mc.Error != nil {
			errs = append(errs, mc.Error)
		}
	}

	if len(errs) > 0 {
		return nil, fmt.Errorf("failed to create multicalls: %v", errors.Join(errs...))
	}

	allMultiCalls := utils.Map(requests, func(mc *MulticallAndError, index uint64) *multicall.MultiCallMetaData[[]common.Address] {
		return mc.Multicall
	})

	client, err := cc.EthereumClient.GetEthereumContractCaller()
	if err != nil {
		cc.Logger.Sugar().Errorw("getOperatorRestakedStrategiesMulticall - failed to get contract caller", zap.Error(err))
		return nil, err
	}

	multicallInstance, err := multicall.NewMulticallClient(ctx, client, &multicall.TMulticallClientOptions{
		MaxBatchSizeBytes: 4096,
		OverrideCallOptions: &bind.CallOpts{
			BlockNumber: big.NewInt(int64(blockNumber)),
		},
		IgnoreErrors: true,
	})
	if err != nil {
		cc.Logger.Sugar().Errorw("getOperatorRestakedStrategiesMulticall - failed to create multicall client", zap.Error(err))
		return nil, err
	}

	results, err := multicall.DoMany(multicallInstance, allMultiCalls...)
	if err != nil {
		cc.Logger.Sugar().Errorw("getOperatorRestakedStrategiesMulticall - failed to execute multicalls", zap.Error(err))
		return nil, err
	}

	if results == nil {
		return nil, errors.New("results are nil")
	}

	mappedResults := utils.Map(*results, func(result *[]common.Address, i uint64) *contractCaller.OperatorRestakedStrategy {
		oas := operatorRestakedStrategies[i]
		if result == nil {
			oas.Results = nil
		} else {
			oas.Results = *result
		}
		return oas
	})

	resultsWithErrors := &RestakedStrategiesWithErrors{
		OperatorRestakedStrategies: make([]*contractCaller.OperatorRestakedStrategy, 0),
		Errors:                     make([]*contractCaller.OperatorRestakedStrategy, 0),
	}
	utils.Reduce(mappedResults, func(acc *RestakedStrategiesWithErrors, ors *contractCaller.OperatorRestakedStrategy) *RestakedStrategiesWithErrors {
		if ors.Results == nil {
			acc.Errors = append(acc.Errors, ors)
		} else {
			acc.OperatorRestakedStrategies = append(acc.OperatorRestakedStrategies, ors)
		}
		return acc
	}, resultsWithErrors)

	if len(resultsWithErrors.Errors) > 0 {
		cc.Logger.Sugar().Errorw("getOperatorRestakedStrategiesMulticall - failed to get results for some operators",
			zap.Int("numErrors", len(resultsWithErrors.Errors)),
		)
		for _, err := range resultsWithErrors.Errors {
			cc.Logger.Sugar().Errorw("getOperatorRestakedStrategiesMulticall - failed to get results for operator",
				zap.String("operator", err.Operator),
				zap.String("avs", err.Avs),
			)
		}
	}
	return resultsWithErrors.OperatorRestakedStrategies, nil
}

type RestakedStrategiesWithErrors struct {
	OperatorRestakedStrategies []*contractCaller.OperatorRestakedStrategy
	Errors                     []*contractCaller.OperatorRestakedStrategy
}

type ReconciledContractCaller struct {
	EthereumClients []*ethereum.Client
	Logger          *zap.Logger
}

func NewRecociledContractCaller(ec []*ethereum.Client, l *zap.Logger) (*ReconciledContractCaller, error) {
	if len(ec) == 0 {
		return nil, errors.New("no ethereum clients provided")
	}
	return &ReconciledContractCaller{
		EthereumClients: ec,
		Logger:          l,
	}, nil
}

func (rcc *ReconciledContractCaller) GetOperatorRestakedStrategies(ctx context.Context, avs string, operator string, blockNumber uint64) ([]common.Address, error) {
	allResults := make([][]common.Address, 0)
	for i, ec := range rcc.EthereumClients {
		results, err := getOperatorRestakedStrategies(ctx, avs, operator, blockNumber, ec, rcc.Logger)
		if err != nil {
			rcc.Logger.Sugar().Errorw("error fetching results for client", zap.Error(err), zap.Int("clientIndex", i))
		} else {
			allResults = append(allResults, results)
		}
	}

	// make sure the number of total results is equal to the number of clients
	if len(allResults) != len(rcc.EthereumClients) {
		return nil, errors.New("failed to fetch results for all clients")
	}

	if len(allResults) == 1 {
		return allResults[0], nil
	}

	// make sure that the results from each client are all the same length
	expectedLength := len(allResults[0])
	for i := 1; i < len(allResults); i++ {
		if len(allResults[i]) != expectedLength {
			return nil, fmt.Errorf("client %d returned unexpected number of results", i)
		}
	}

	// check each item in each result to make sure they are all the same
	for _, clientResult := range allResults[1:] {
		for i, item := range clientResult {
			if allResults[0][i] != item {
				return nil, errors.New("client results do not match")
			}
		}
	}

	return allResults[0], nil
}

func (cc *MulticallContractCaller) GetDistributionRootByIndex(ctx context.Context, index uint64) (*IRewardsCoordinator.IRewardsCoordinatorDistributionRoot, error) {
	return nil, errors.New("not implemented")
}
