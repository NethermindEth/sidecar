package pipeline

import (
	"context"
	"github.com/Layr-Labs/sidecar/internal/fetcher"
	"github.com/Layr-Labs/sidecar/internal/indexer"
	"go.uber.org/zap"
)

type Pipeline struct {
	Fetcher *fetcher.Fetcher
	Indexer *indexer.Indexer
	Logger  *zap.Logger
}

func NewPipeline(f *fetcher.Fetcher, i *indexer.Indexer, l *zap.Logger) *Pipeline {
	return &Pipeline{
		Fetcher: f,
		Indexer: i,
		Logger:  l,
	}
}

func (p *Pipeline) RunForBlock(ctx context.Context, blockNumber uint64) error {
	p.Logger.Sugar().Infow("Running pipeline for block", zap.Uint64("blockNumber", blockNumber))

	/*
		- Fetch block
		- Index block
		- Index transactions
		- Index logs
		- Check for contract upgrades
	*/

	block, err := p.Fetcher.FetchBlock(ctx, blockNumber)
	if err != nil {
		p.Logger.Sugar().Errorw("Failed to fetch block", zap.Uint64("blockNumber", blockNumber), zap.Error(err))
		return err
	}

	indexedBlock, found, err := p.Indexer.IndexFetchedBlock(block)
	if err != nil {
		p.Logger.Sugar().Errorw("Failed to index block", zap.Uint64("blockNumber", blockNumber), zap.Error(err))
		return err
	}
	if found {
		p.Logger.Sugar().Infow("Block already indexed", zap.Uint64("blockNumber", blockNumber))
	}

	// We can set as-batch to true since none of these transactions should exist already
	indexedTransactions, err := p.Indexer.IndexTransactions(ctx, indexedBlock, block, true)
	if err != nil {
		p.Logger.Sugar().Errorw("Failed to index transactions", zap.Uint64("blockNumber", blockNumber), zap.Error(err))
		return err
	}
	p.Logger.Sugar().Infow("Indexed transactions", zap.Uint64("blockNumber", blockNumber), zap.Int("count", len(indexedTransactions)))

	p.Indexer.FindAndHandleContractCreationForTransactions(block.Block.Transactions, block.TxReceipts, block.ContractStorage, blockNumber)

	return nil
}
