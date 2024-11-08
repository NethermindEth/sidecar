package fetcher

import (
	"context"
	"github.com/Layr-Labs/go-sidecar/internal/config"
	"github.com/Layr-Labs/go-sidecar/pkg/clients/ethereum"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"slices"
	"sync"
	"time"
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

	receipts, err := f.FetchReceiptsForBlock(ctx, block)
	if err != nil {
		f.Logger.Sugar().Errorw("failed to fetch receipts for block", zap.Error(err))
		return nil, err
	}

	return &FetchedBlock{
		Block:      block,
		TxReceipts: receipts,
	}, nil
}

func (f *Fetcher) FetchReceiptsForBlock(ctx context.Context, block *ethereum.EthereumBlock) (map[string]*ethereum.EthereumTransactionReceipt, error) {
	blockNumber := block.Number.Value()

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
	return receipts, nil
}

func (f *Fetcher) IsInterestingAddress(contractAddress string) bool {
	return slices.Contains(f.Config.GetInterestingAddressForConfigEnv(), contractAddress)
}

func (f *Fetcher) FetchBlocksWithRetries(ctx context.Context, startBlockInclusive uint64, endBlockInclusive uint64) ([]*FetchedBlock, error) {
	retries := []int{1, 2, 4, 8, 16, 32, 64}
	var e error
	for _, r := range retries {
		fetchedBlocks, err := f.FetchBlocks(ctx, startBlockInclusive, endBlockInclusive)
		if err == nil {
			return fetchedBlocks, nil
		}
		e = err
		f.Logger.Sugar().Infow("failed to fetch blocks for range",
			zap.Uint64("startBlock", startBlockInclusive),
			zap.Uint64("endBlock", endBlockInclusive),
			zap.Int("sleepTime", r),
		)

		time.Sleep(time.Duration(r) * time.Second)
	}
	f.Logger.Sugar().Errorw("failed to fetch blocks for range, exhausted all retries",
		zap.Uint64("startBlock", startBlockInclusive),
		zap.Uint64("endBlock", endBlockInclusive),
		zap.Error(e),
	)
	return nil, e
}

func (f *Fetcher) FetchBlocks(ctx context.Context, startBlockInclusive uint64, endBlockInclusive uint64) ([]*FetchedBlock, error) {
	blockNumbers := make([]uint64, 0)
	for i := startBlockInclusive; i <= endBlockInclusive; i++ {
		blockNumbers = append(blockNumbers, i)
	}

	blockRequests := make([]*ethereum.RPCRequest, 0)
	for i, n := range blockNumbers {
		blockRequests = append(blockRequests, ethereum.GetBlockByNumberRequest(n, uint(i)))
	}

	blockResponses, err := f.EthClient.BatchCall(ctx, blockRequests)
	if err != nil {
		f.Logger.Sugar().Errorw("failed to batch call for blocks", zap.Error(err))
		return nil, err
	}

	blocks := make([]*ethereum.EthereumBlock, 0)
	for _, response := range blockResponses {
		b, err := ethereum.RPCMethod_getBlockByNumber.ResponseParser(response.Result)
		if err != nil {
			f.Logger.Sugar().Errorw("failed to parse block",
				zap.Error(err),
				zap.Uint("response ID", *response.ID),
			)
			return nil, err
		}
		blocks = append(blocks, b)
	}
	if len(blocks) != len(blockNumbers) {
		f.Logger.Sugar().Errorw("failed to fetch all blocks",
			zap.Int("fetched", len(blocks)),
			zap.Int("expected", len(blockNumbers)),
		)
		return nil, err
	}

	fetchedBlocks := make([]*FetchedBlock, 0)
	foundErrors := false
	wg := sync.WaitGroup{}
	for _, block := range blocks {
		wg.Add(1)
		go func(b *ethereum.EthereumBlock) {
			defer wg.Done()
			receipts, err := f.FetchReceiptsForBlock(ctx, b)
			if err != nil {
				f.Logger.Sugar().Errorw("failed to fetch receipts for block",
					zap.Uint64("blockNumber", b.Number.Value()),
					zap.Error(err),
				)
				foundErrors = true
				return
			}
			fetchedBlocks = append(fetchedBlocks, &FetchedBlock{
				Block:      b,
				TxReceipts: receipts,
			})
		}(block)
	}
	wg.Wait()
	if foundErrors {
		return nil, errors.New("failed to fetch receipts for some blocks")
	}
	if len(fetchedBlocks) != len(blocks) {
		f.Logger.Sugar().Errorw("failed to fetch all blocks",
			zap.Int("fetched", len(fetchedBlocks)),
			zap.Int("expected", len(blocks)),
		)
		return nil, errors.New("failed to fetch all blocks")
	}

	f.Logger.Sugar().Debugw("Fetched blocks",
		zap.Int("count", len(fetchedBlocks)),
		zap.Uint64("startBlock", startBlockInclusive),
		zap.Uint64("endBlock", endBlockInclusive),
	)

	return fetchedBlocks, nil
}
