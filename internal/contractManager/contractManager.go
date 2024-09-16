package contractManager

import (
	"context"
	"fmt"

	"github.com/DataDog/datadog-go/v5/statsd"
	"github.com/Layr-Labs/go-sidecar/internal/clients/ethereum"
	"github.com/Layr-Labs/go-sidecar/internal/clients/etherscan"
	"github.com/Layr-Labs/go-sidecar/internal/contractStore"
	"github.com/Layr-Labs/go-sidecar/internal/metrics"
	"github.com/Layr-Labs/go-sidecar/internal/utils"
	"go.uber.org/zap"
)

type ContractManager struct {
	ContractStore   contractStore.ContractStore
	EtherscanClient *etherscan.EtherscanClient
	EthereumClient  *ethereum.Client
	Statsd          *statsd.Client
	Logger          *zap.Logger
}

func NewContractManager(
	cs contractStore.ContractStore,
	ec *etherscan.EtherscanClient,
	e *ethereum.Client,
	s *statsd.Client,
	l *zap.Logger,
) *ContractManager {
	return &ContractManager{
		ContractStore:   cs,
		EtherscanClient: ec,
		EthereumClient:  e,
		Statsd:          s,
		Logger:          l,
	}
}

func (cm *ContractManager) FindOrCreateContractWithProxy(
	contractAddress string,
	blockNumber uint64,
	bytecodeHash string,
	reindexContract bool,
) (*contractStore.ContractsTree, error) {
	cm.Logger.Sugar().Debugw(fmt.Sprintf("Getting contract for address '%s'", contractAddress))

	contract, err := cm.ContractStore.GetContractWithProxyContract(contractAddress, blockNumber)
	if err != nil {
		cm.Logger.Sugar().Errorw("Failed to get contract for address", zap.Error(err), zap.String("contractAddress", contractAddress))
		return nil, err
	}

	// Contract found, return it
	// NOTE: at this point, if a contract is in the database, it's safe to assume we've
	// tried to fetch the ABI for it and tried to determine if its a proxy contract.
	if contract != nil && !reindexContract {
		cm.Logger.Sugar().Debugw(fmt.Sprintf("Found contract in database '%s'", contractAddress))
		return contract, nil
	}

	if bytecodeHash == "" {
		cm.Logger.Sugar().Debugw(fmt.Sprintf("No bytecode hash provided for contract '%s'", contractAddress), zap.String("contractAddress", contractAddress))
	}

	_, err = cm.CreateContract(contractAddress, bytecodeHash, reindexContract)
	if err != nil {
		cm.Logger.Sugar().Errorw("Failed to create contract", zap.Error(err), zap.String("contractAddress", contractAddress))
		return nil, err
	}

	storageValue, err := cm.EthereumClient.GetStorageAt(context.Background(), contractAddress, ethereum.EIP1967_STORAGE_SLOT, "latest")
	if err != nil {
		cm.Logger.Sugar().Errorw("Failed to get storage value", zap.Error(err), zap.String("contractAddress", contractAddress))
	} else {
		cm.HandleProxyContractCreation(contractAddress, storageValue, blockNumber, reindexContract)
	}

	// Try to re-fetch contract
	return cm.ContractStore.GetContractWithProxyContract(contractAddress, blockNumber)
}

func (cm *ContractManager) CreateContract(
	contractAddress string,
	bytecodeHash string,
	reindexContract bool,
) (*contractStore.Contract, error) {
	// If the bytecode hash wasnt provided, fetch it
	if bytecodeHash == "" {
		cm.Logger.Sugar().Debugw("No bytecode hash provided for contract, fetching",
			zap.String("contractAddress", contractAddress),
		)
		bytecode, err := cm.EthereumClient.GetCode(context.Background(), contractAddress)
		if err != nil {
			cm.Logger.Sugar().Errorw("Failed to get contract bytecode",
				zap.Error(err),
				zap.String("contractAddress", contractAddress),
			)
		} else {
			bytecodeHash = ethereum.HashBytecode(bytecode)
			cm.Logger.Sugar().Debugw("Fetched contract bytecode",
				zap.String("contractAddress", contractAddress),
				zap.String("bytecodeHash", bytecodeHash),
			)
		}
	}

	// Record the contract in the contracts table
	_, _, err := cm.ContractStore.FindOrCreateContract(
		contractAddress,
		"",
		false,
		bytecodeHash,
		"",
		false,
	)
	if err != nil {
		cm.Logger.Sugar().Errorw("Failed to create new contract",
			zap.Error(err),
			zap.String("contractAddress", contractAddress),
		)
	} else {
		cm.Logger.Sugar().Debugf(fmt.Sprintf("Created new contract '%s'", contractAddress))
	}

	contract := cm.FindAndSetContractAbi(contractAddress)

	cm.FindAndSetLookalikeContract(contractAddress, bytecodeHash)

	return contract, nil
}

func (cm *ContractManager) FindAndSetLookalikeContract(
	contractAddress string,
	bytecodeHash string,
) {
	// Attempt to find a lookalike contract by comparing the bytecode hash with contracts that already exist
	similar, err := cm.ContractStore.FindVerifiedContractWithMatchingBytecodeHash(bytecodeHash, contractAddress)
	if err != nil {
		cm.Logger.Sugar().Errorw("Failed to find similar contract",
			zap.Error(err),
			zap.String("contractAddress", contractAddress),
		)
	}
	if similar == nil {
		cm.Logger.Sugar().Debugw("No similar contract found",
			zap.String("contractAddress", contractAddress),
		)
	} else {
		cm.Logger.Sugar().Debugw("Found similar contract",
			zap.String("contractAddress", contractAddress),
			zap.String("similarContractAddress", similar.ContractAddress),
		)
		_, err := cm.ContractStore.SetContractMatchingContractAddress(contractAddress, similar.ContractAddress)
		if err != nil {
			cm.Logger.Sugar().Errorw("Failed to update contract ABI",
				zap.Error(err),
				zap.String("contractAddress", contractAddress),
			)
		} else {
			cm.Logger.Sugar().Debugw("Updated contract with similar contract",
				zap.String("contractAddress", contractAddress),
				zap.String("similarContractAddress", similar.ContractAddress),
			)
		}
	}
}

func (cm *ContractManager) FindAndSetContractAbi(contractAddress string) *contractStore.Contract {
	// Attempt to find the ABI for the contract on Etherscan
	err := cm.Statsd.Incr(metrics.Etherscan_ContractAbi, nil, 1)
	if err != nil {
		cm.Logger.Sugar().Warnw("Failed to increment metric",
			zap.Error(err),
			zap.String("metric", metrics.Etherscan_ContractAbi),
		)
	}

	abi, err := cm.EtherscanClient.ContractABI(contractAddress)
	if err != nil {
		cm.Logger.Sugar().Errorw("Failed to update contract ABI",
			zap.Error(err),
			zap.String("contractAddress", contractAddress),
		)
	}

	if err != nil {
		cm.Logger.Sugar().Debugw(fmt.Sprintf("Failed to get contract ABI from etherscan - '%s'", contractAddress),
			zap.Error(err),
			zap.String("contractAddress", contractAddress),
		)
	}

	if abi == "" {
		cm.Logger.Sugar().Debugw(fmt.Sprintf("ABI received from etherscan is empty - '%s'", contractAddress),
			zap.String("contractAddress", contractAddress),
		)
	}

	// Update contract ABI and mark the contract as having checked for the ABI
	c, err := cm.ContractStore.SetContractAbi(contractAddress, abi, abi != "")
	if err != nil {
		cm.Logger.Sugar().Errorw("Failed to set contract ABI",
			zap.Error(err),
			zap.String("contractAddress", contractAddress),
		)
	}
	return c
}

func (cm *ContractManager) CreateProxyContract(
	contractAddress string,
	proxyContractAddress string,
	blockNumber uint64,
	reindexContract bool,
) (*contractStore.ProxyContract, error) {
	proxyContract, found, err := cm.ContractStore.FindOrCreateProxyContract(blockNumber, contractAddress, proxyContractAddress)
	if err != nil {
		cm.Logger.Sugar().Errorw("Failed to create proxy contract",
			zap.Error(err),
			zap.String("contractAddress", contractAddress),
			zap.String("proxyContractAddress", proxyContractAddress),
		)
	} else {
		if found {
			cm.Logger.Sugar().Debugw("Found existing proxy contract",
				zap.String("contractAddress", contractAddress),
				zap.String("proxyContractAddress", proxyContractAddress),
			)
		} else {
			cm.Logger.Sugar().Debugw("Created proxy contract",
				zap.String("contractAddress", contractAddress),
				zap.String("proxyContractAddress", proxyContractAddress),
			)
		}
	}
	// Check to see if the contract we're proxying to is already in the database
	proxiedContract, err := cm.ContractStore.GetContractForAddress(proxyContractAddress)
	if err != nil {
		cm.Logger.Sugar().Errorw("Failed to get contract for address",
			zap.Error(err),
			zap.String("contractAddress", proxyContractAddress),
		)
	}
	if proxiedContract != nil {
		cm.Logger.Sugar().Debugw("Found proxied contract",
			zap.String("contractAddress", proxyContractAddress),
			zap.String("proxiedContractAddress", proxiedContract.ContractAddress),
		)
	} else {
		_, err := cm.CreateContract(proxyContractAddress, "", reindexContract)
		if err != nil {
			cm.Logger.Sugar().Errorw("Failed to create contract",
				zap.Error(err),
				zap.String("contractAddress", proxyContractAddress),
			)
		} else {
			cm.Logger.Sugar().Debugw("Created contract",
				zap.String("contractAddress", proxyContractAddress),
			)
		}
	}
	return proxyContract, nil
}

func (cm *ContractManager) HandleProxyContractCreation(
	contractAddress string,
	eip1197StoredValue string,
	blockNumber uint64,
	reindexContract bool,
) {
	// Determine if the contract is a proxy contract
	// 0x + 64 char hex string
	if len(eip1197StoredValue) != 66 {
		cm.Logger.Sugar().Debugw("Stored data is not a proxy contract",
			zap.String("contractAddress", contractAddress),
			zap.String("storedData", eip1197StoredValue),
		)
		_, err := cm.ContractStore.SetContractCheckedForProxy(contractAddress)
		if err != nil {
			cm.Logger.Sugar().Errorw("Failed to set contract as checked for proxy",
				zap.Error(err),
				zap.String("contractAddress", contractAddress),
			)
		}
		return
	}
	proxyContractAddress := fmt.Sprintf("0x%s", eip1197StoredValue[26:])

	if len(proxyContractAddress) != 42 || proxyContractAddress == utils.NullEthereumAddressHex {
		cm.Logger.Sugar().Debugw("Stored address is either null or invalid",
			zap.String("contractAddress", contractAddress),
			zap.String("storedData", eip1197StoredValue),
		)
		return
	}

	_, err := cm.CreateProxyContract(contractAddress, proxyContractAddress, blockNumber, reindexContract)
	if err != nil {
		cm.Logger.Sugar().Errorw("Failed to create proxy contract",
			zap.Error(err),
			zap.String("contractAddress", contractAddress),
			zap.String("proxyContractAddress", proxyContractAddress),
		)
	}
}
