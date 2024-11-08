package sequentialContractCaller

import (
	"context"
	"github.com/Layr-Labs/go-sidecar/pkg/clients/ethereum"
	"github.com/Layr-Labs/go-sidecar/pkg/contractCaller"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"go.uber.org/zap"
	"math/big"
	"regexp"
	"strings"
	"time"
)

type SequentialContractCaller struct {
	EthereumClient *ethereum.Client
	Logger         *zap.Logger
}

func NewSequentialContractCaller(ec *ethereum.Client, l *zap.Logger) *SequentialContractCaller {
	return &SequentialContractCaller{
		EthereumClient: ec,
		Logger:         l,
	}
}

func isExecutionRevertedError(err error) bool {
	r := regexp.MustCompile(`execution reverted`)
	return r.MatchString(err.Error())
}

func getOperatorRestakedStrategiesRetryable(ctx context.Context, avs string, operator string, blockNumber uint64, client *ethereum.Client, l *zap.Logger) ([]common.Address, error) {
	retries := []int{0, 2, 5, 10}
	for i, backoff := range retries {
		results, err := getOperatorRestakedStrategies(ctx, avs, operator, blockNumber, client, l)
		if err != nil {
			l.Sugar().Errorw("GetOperatorRestakedStrategiesRetryable - failed to get results",
				zap.Int("attempt", i+1),
				zap.String("avs", avs),
				zap.String("operator", operator),
				zap.Uint64("blockNumber", blockNumber),
				zap.Error(err),
			)
			time.Sleep(time.Second * time.Duration(backoff))
		} else {
			return results, nil
		}
	}
	return nil, nil
}

func getOperatorRestakedStrategies(ctx context.Context, avs string, operator string, blockNumber uint64, client *ethereum.Client, l *zap.Logger) ([]common.Address, error) {
	a, err := abi.JSON(strings.NewReader(contractCaller.AvsServiceManagerAbi))
	if err != nil {
		l.Sugar().Errorw("GetOperatorRestakedStrategies - failed to parse abi", zap.Error(err))
		return nil, err
	}

	callerClient, err := client.GetEthereumContractCaller()
	if err != nil {
		l.Sugar().Errorw("GetOperatorRestakedStrategies - failed to get contract caller", zap.Error(err))
		return nil, err
	}

	contract := bind.NewBoundContract(common.HexToAddress(avs), a, callerClient, nil, nil)

	bigBlockNumber := big.NewInt(int64(blockNumber))

	results := make([]interface{}, 0)

	err = contract.Call(&bind.CallOpts{BlockNumber: bigBlockNumber, Context: ctx}, &results, "getOperatorRestakedStrategies", common.HexToAddress(operator))
	if err != nil {
		if isExecutionRevertedError(err) {
			return nil, nil
		}
		return nil, err
	}

	return results[0].([]common.Address), nil
}

func (cc *SequentialContractCaller) GetOperatorRestakedStrategies(ctx context.Context, avs string, operator string, blockNumber uint64) ([]common.Address, error) {
	return getOperatorRestakedStrategiesRetryable(ctx, avs, operator, blockNumber, cc.EthereumClient, cc.Logger)
}
