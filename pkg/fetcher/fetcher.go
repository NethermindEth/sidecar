package fetcher

import (
	"context"
	"github.com/Layr-Labs/go-sidecar/internal/config"
	"github.com/Layr-Labs/go-sidecar/pkg/clients/ethereum"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"go.uber.org/zap"
	"slices"
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

	return &FetchedBlock{
		Block:      block,
		TxReceipts: receipts,
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
