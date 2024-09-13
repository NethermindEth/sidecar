package fetcher

import (
	"context"
	"slices"
	"sync"

	"github.com/Layr-Labs/go-sidecar/internal/clients/ethereum"
	"github.com/Layr-Labs/go-sidecar/internal/config"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"go.uber.org/zap"
)

type Fetcher struct {
	EthClient *ethereum.Client
	Logger    *zap.Logger
	Config    *config.Config
}

func NewFetcher(ethClient *ethereum.Client, cfg *config.Config, l *zap.Logger) *Fetcher {
	return &Fetcher{
		EthClient: ethClient,
		Logger:    l,
		Config:    cfg,
	}
}

type FetchedBlock struct {
	Block *ethereum.EthereumBlock
	// map[transactionHash] => transactionReceipt
	TxReceipts map[string]*ethereum.EthereumTransactionReceipt
	// map[contractAddress] => stored value
	ContractStorage map[string]string
}

func (f *Fetcher) FetchBlock(ctx context.Context, blockNumber uint64) (*FetchedBlock, error) {
	block, err := f.EthClient.GetBlockByNumber(ctx, blockNumber)
	if err != nil {
		f.Logger.Sugar().Errorw("failed to get block by number", zap.Error(err))
		return nil, err
	}

	txReceiptRequests := make([]*ethereum.RPCRequest, 0)
	f.Logger.Sugar().Debugf("Fetching '%d' transactions from block '%d'", len(block.Transactions), blockNumber)

	for i, tx := range block.Transactions {
		txReceiptRequests = append(txReceiptRequests, ethereum.GetTransactionReceiptRequest(tx.Hash.Value(), uint(i)))
	}

	f.Logger.Sugar().Debugw("Fetching transaction receipts",
		zap.Int("count", len(txReceiptRequests)),
		zap.Uint64("blockNumber", blockNumber),
	)

	receiptResponses, err := f.EthClient.BatchCall(ctx, txReceiptRequests)
	if err != nil {
		f.Logger.Sugar().Errorw("failed to batch call for transaction receipts", zap.Error(err))
		return nil, err
	}

	receipts := make(map[string]*ethereum.EthereumTransactionReceipt)
	for _, response := range receiptResponses {
		r, err := ethereum.RPCMethod_getTransactionReceipt.ResponseParser(response.Result)
		if err != nil {
			f.Logger.Sugar().Errorw("failed to parse transaction receipt",
				zap.Error(err),
				zap.Uint("response ID", *response.ID),
			)
			return nil, err
		}
		receipts[r.TransactionHash.Value()] = r
	}

	// Use a map to get only unique contract addresses
	createdContractMap := make(map[string]bool, 0)
	for _, r := range receipts {
		if r.To == "" && r.ContractAddress != "" && f.IsInterestingAddress(r.ContractAddress.Value()) {
			createdContractMap[r.ContractAddress.Value()] = true
		}
	}

	// Convert keys back into a list
	createdContracts := make([]string, 0)
	for k := range createdContractMap {
		createdContracts = append(createdContracts, k)
	}

	contractStorage := make(map[string]string)

	// address -> bytecode
	contractBytecodeMap := make(map[string]string)

	mapMutex := sync.Mutex{}

	if len(createdContracts) > 0 {
		wg := sync.WaitGroup{}
		for _, contractAddress := range createdContracts {
			wg.Add(1)
			go func(contractAddress string) {
				defer wg.Done()
				// block 0 implies latest block
				storageValue, err := f.GetContractStorageSlot(ctx, contractAddress, 0)
				if err != nil {
					f.Logger.Sugar().Errorw("failed to get storage value",
						zap.Error(err),
						zap.String("contractAddress", contractAddress),
					)
				} else {
					f.Logger.Sugar().Debugw("Fetched storage value",
						zap.String("contractAddress", contractAddress),
						zap.String("storageValue", storageValue),
					)
					mapMutex.Lock()
					contractStorage[contractAddress] = storageValue
					mapMutex.Unlock()
				}

				contractBytecode, err := f.EthClient.GetCode(ctx, contractAddress)
				if err != nil {
					f.Logger.Sugar().Errorw("failed to get contract bytecode",
						zap.Error(err),
						zap.String("contractAddress", contractAddress),
					)
				} else {
					mapMutex.Lock()
					contractBytecodeMap[contractAddress] = contractBytecode
					mapMutex.Unlock()
				}
			}(contractAddress)
		}
		wg.Wait()
		// Attach bytecode to receipt
		for _, receipt := range receipts {
			if receipt.ContractAddress != "" {
				if cb, ok := contractBytecodeMap[receipt.ContractAddress.Value()]; ok {
					receipt.ContractBytecode = ethereum.EthereumHexString(cb)
				}
			}
		}
	}

	return &FetchedBlock{
		Block:           block,
		TxReceipts:      receipts,
		ContractStorage: contractStorage,
	}, nil
}

func (f *Fetcher) IsInterestingAddress(contractAddress string) bool {
	return slices.Contains(f.Config.GetInterestingAddressForConfigEnv(), contractAddress)
}

func (f *Fetcher) GetContractStorageSlot(ctx context.Context, contractAddress string, blockNumber uint64) (string, error) {
	stringBlock := ""
	if blockNumber == 0 {
		stringBlock = "latest"
	} else {
		stringBlock = hexutil.EncodeUint64(blockNumber)
	}

	return f.EthClient.GetStorageAt(ctx, contractAddress, ethereum.EIP1967_STORAGE_SLOT, stringBlock)
}
