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
	"sync"
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
	retries := []int{0, 2, 5, 10, 30}
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

func (cc *SequentialContractCaller) getOperatorRestakedStrategiesBatch(ctx context.Context, operatorRestakedStrategies []*contractCaller.OperatorRestakedStrategy, blockNumber uint64) ([]*contractCaller.OperatorRestakedStrategy, error) {
	var wg sync.WaitGroup
	responses := make(chan *contractCaller.OperatorRestakedStrategy, len(operatorRestakedStrategies))

	for _, operatorRestakedStrategy := range operatorRestakedStrategies {
		wg.Add(1)
		// make a local copy of the entire struct
		currentReq := *operatorRestakedStrategy
		go func() {
			defer wg.Done()
			results, err := getOperatorRestakedStrategiesRetryable(ctx, currentReq.Avs, currentReq.Operator, blockNumber, cc.EthereumClient, cc.Logger)
			if err != nil {
				cc.Logger.Sugar().Errorw("getOperatorRestakedStrategiesBatch - failed to get results",
					zap.String("avs", currentReq.Avs),
					zap.String("operator", currentReq.Operator),
					zap.Uint64("blockNumber", blockNumber),
					zap.Error(err),
				)
				return
			}
			currentReq.Results = results
			
			// send back a pointer to the copied struct
			responses <- &currentReq
		}()
	}
	wg.Wait()
	close(responses)

	allResponses := make([]*contractCaller.OperatorRestakedStrategy, 0)
	for response := range responses {
		allResponses = append(allResponses, response)
	}
	return allResponses, nil
}

const BATCH_SIZE = 25

func (cc *SequentialContractCaller) GetAllOperatorRestakedStrategies(ctx context.Context, operatorRestakedStrategies []*contractCaller.OperatorRestakedStrategy, blockNumber uint64) ([]*contractCaller.OperatorRestakedStrategy, error) {
	batches := make([][]*contractCaller.OperatorRestakedStrategy, 0)
	currentIndex := 0
	for {
		endIndex := currentIndex + BATCH_SIZE
		if endIndex >= len(operatorRestakedStrategies) {
			endIndex = len(operatorRestakedStrategies)
		}
		batches = append(batches, operatorRestakedStrategies[currentIndex:endIndex])
		currentIndex = currentIndex + BATCH_SIZE
		if currentIndex >= len(operatorRestakedStrategies) {
			break
		}
	}
	cc.Logger.Sugar().Infow("GetAllOperatorRestakedStrategies - batches",
		zap.Int("batches", len(batches)),
		zap.Int("total", len(operatorRestakedStrategies)),
		zap.Uint64("blockNumber", blockNumber),
	)

	allResults := make([]*contractCaller.OperatorRestakedStrategy, 0)
	for _, batch := range batches {
		results, err := cc.getOperatorRestakedStrategiesBatch(ctx, batch, blockNumber)
		if err != nil {
			return nil, err
		}
		allResults = append(allResults, results...)
	}
	return allResults, nil
}
