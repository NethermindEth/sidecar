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

	// Parse all transactions and logs for the block.
	// - If a transaction is not calling to a contract, it is ignored
	// - If a transaction has 0 interesting logs and itself is not interesting, it is ignored
	parsedTransactions, ierr := p.Indexer.ParseInterestingTransactionsAndLogs(ctx, block)
	if ierr != nil {
		p.Logger.Sugar().Errorw("Failed to parse transactions and logs",
			zap.Uint64("blockNumber", blockNumber),
			zap.String("transactionHash", ierr.TransactionHash),
			zap.Error(ierr.Err),
		)
		return err
	}
	p.Logger.Sugar().Infow("Parsed transactions", zap.Uint64("blockNumber", blockNumber), zap.Int("count", len(parsedTransactions)))

	// With only interesting transactions/logs parsed, insert them into the database
	for _, pt := range parsedTransactions {
		indexedTransaction, err := p.Indexer.IndexTransaction(indexedBlock, pt.Transaction, pt.Receipt)
		if err != nil {
			p.Logger.Sugar().Errorw("Failed to index transaction",
				zap.Uint64("blockNumber", blockNumber),
				zap.String("transactionHash", pt.Transaction.Hash.Value()),
				zap.Error(err),
			)
			return err
		}
		p.Logger.Debug("Indexed transaction", zap.Uint64("blockNumber", blockNumber), zap.String("transactionHash", indexedTransaction.TransactionHash))

		for _, log := range pt.Logs {
			_, err := p.Indexer.IndexLog(
				ctx,
				indexedBlock.Number,
				indexedBlock.Id,
				indexedTransaction.TransactionHash,
				indexedTransaction.TransactionIndex,
				log,
			)
			if err != nil {
				p.Logger.Sugar().Errorw("Failed to index log",
					zap.Uint64("blockNumber", blockNumber),
					zap.String("transactionHash", pt.Transaction.Hash.Value()),
					zap.Uint64("logIndex", log.LogIndex),
					zap.Error(err),
				)
				return err
			}
			p.Logger.Sugar().Debugw("Indexed log",
				zap.Uint64("blockNumber", blockNumber),
				zap.String("transactionHash", indexedTransaction.TransactionHash),
				zap.Uint64("logIndex", log.LogIndex),
			)
		}
		// Check the logs for any contract upgrades.
		upgradedLogs := p.Indexer.FindContractUpgradedLogs(pt.Logs)
		if len(upgradedLogs) > 0 {
			p.Logger.Sugar().Debugw("Found contract upgrade logs",
				zap.String("txHash", pt.Transaction.Hash.Value()),
				zap.Uint64("block", pt.Transaction.BlockNumber.Value()),
				zap.Int("count", len(upgradedLogs)),
			)

			p.Indexer.IndexContractUpgrades(block.Block.Number.Value(), upgradedLogs, false)
		}
	}

	// Handle contract creation for transactions
	interestingTransactions := p.Indexer.FilterInterestingTransactions(indexedBlock, block)
	if len(interestingTransactions) == 0 {
		p.Logger.Sugar().Debugw("No interesting transactions found, no need to create any new contracts", zap.Uint64("blockNumber", blockNumber))
		return nil
	}
	p.Indexer.FindAndHandleContractCreationForTransactions(interestingTransactions, block.TxReceipts, block.ContractStorage, blockNumber)

	return nil
}

func (p *Pipeline) CalculateSomething() {

}
