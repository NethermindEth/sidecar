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
	EthereumClient *ethereum.Client
	httpClient     *http.Client
	Logger         *zap.Logger
	Config         *config.Config
	AbiSources     []abiSource.AbiSource
}

func NewAbiFetcher(
	e *ethereum.Client,
	hc *http.Client,
	l *zap.Logger,
	cfg *config.Config,
	sources []abiSource.AbiSource,
) *AbiFetcher {
	return &AbiFetcher{
		EthereumClient: e,
		httpClient:     hc,
		Logger:         l,
		Config:         cfg,
		AbiSources:     sources,
	}
}

func (af *AbiFetcher) FetchContractDetails(ctx context.Context, address string) (string, string, error) {
	bytecode, err := af.EthereumClient.GetCode(ctx, address)
	if err != nil {
		af.Logger.Sugar().Errorw("Failed to get the contract bytecode",
			zap.Error(err),
			zap.String("address", address),
		)
		return "", "", err
	}

	bytecodeHash := ethereum.HashBytecode(bytecode)
	af.Logger.Sugar().Debug("Fetched the contract bytecodeHash",
		zap.String("address", address),
		zap.String("bytecodeHash", bytecodeHash),
	)

	// fetch ABI in order of AbiSource implementations
	for _, source := range af.AbiSources {
		abi, err := source.FetchAbi(address, bytecode)
		if err == nil {
			return bytecodeHash, abi, nil
		}
	}
	return "", "", fmt.Errorf("failed to fetch ABI for contract %s", address)
}
