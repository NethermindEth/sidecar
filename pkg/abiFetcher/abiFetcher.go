package abiFetcher

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Layr-Labs/sidecar/internal/config"
	"github.com/Layr-Labs/sidecar/pkg/abiSource"
	"github.com/Layr-Labs/sidecar/pkg/clients/ethereum"
	"go.uber.org/zap"
)

type AbiFetcher struct {
	ethereumClient *ethereum.Client
	httpClient     *http.Client
	logger         *zap.Logger
	config         *config.Config
	abiSources     []abiSource.AbiSource
}

func NewAbiFetcher(
	e *ethereum.Client,
	hc *http.Client,
	l *zap.Logger,
	cfg *config.Config,
	sources []abiSource.AbiSource,
) *AbiFetcher {
	return &AbiFetcher{
		ethereumClient: e,
		httpClient:     hc,
		logger:         l,
		config:         cfg,
		abiSources:     sources,
	}
}

func (af *AbiFetcher) FetchContractDetails(ctx context.Context, address string) (string, string, error) {
	bytecode, err := af.ethereumClient.GetCode(ctx, address)
	if err != nil {
		af.logger.Sugar().Errorw("Failed to get the contract bytecode",
			zap.Error(err),
			zap.String("address", address),
		)
		return "", "", err
	}

	bytecodeHash := ethereum.HashBytecode(bytecode)
	af.logger.Sugar().Debug("Fetched the contract bytecodeHash",
		zap.String("address", address),
		zap.String("bytecodeHash", bytecodeHash),
	)

	// fetch ABI in order of AbiSource implementations
	for _, source := range af.abiSources {
		abi, err := source.FetchAbi(address, bytecode)
		if err == nil {
			af.logger.Sugar().Infow("Successfully fetched ABI",
				zap.String("address", address),
			)
			return bytecodeHash, abi, nil
		}
	}
	return "", "", fmt.Errorf("failed to fetch ABI for contract %s", address)
}
