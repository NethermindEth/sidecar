package etherscan

import (
	"github.com/Layr-Labs/sidecar/pkg/clients/etherscan"
	"go.uber.org/zap"
)

type Etherscan struct {
	etherscanClient *etherscan.EtherscanClient
	logger          *zap.Logger
}

func NewEtherscan(ec *etherscan.EtherscanClient, l *zap.Logger) *Etherscan {
	return &Etherscan{
		etherscanClient: ec,
		logger:          l,
	}
}

func (eas *Etherscan) FetchAbi(address string, bytecode string) (string, error) {
	abi, err := eas.etherscanClient.ContractABI(address)
	if err != nil {
		eas.logger.Sugar().Errorw("Failed to fetch ABI from Etherscan",
			zap.Error(err),
			zap.String("address", address),
		)
		return "", err
	}

	eas.logger.Sugar().Infow("Successfully fetched ABI from Etherscan",
		zap.String("address", address),
	)

	return abi, nil
}
