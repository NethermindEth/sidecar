package etherscanAbiSource

import (
	"github.com/Layr-Labs/sidecar/pkg/clients/etherscan"
	"go.uber.org/zap"
)

type EtherscanAbiSource struct {
	EtherscanClient *etherscan.EtherscanClient
	Logger          *zap.Logger
}

func NewEtherscanAbiSource(ec *etherscan.EtherscanClient, l *zap.Logger) *EtherscanAbiSource {
	return &EtherscanAbiSource{
		EtherscanClient: ec,
		Logger:          l,
	}
}

func (eas *EtherscanAbiSource) FetchAbi(address string, bytecode string) (string, error) {
	abi, err := eas.EtherscanClient.ContractABI(address)
	if err != nil {
		eas.Logger.Sugar().Errorw("Failed to fetch ABI from Etherscan",
			zap.Error(err),
			zap.String("address", address),
		)
		return "", err
	}

	eas.Logger.Sugar().Infow("Successfully fetched ABI from Etherscan",
		zap.String("address", address),
	)

	return abi, nil
}
