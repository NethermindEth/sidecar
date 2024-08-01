package indexer

import (
	"context"
	"fmt"
	"github.com/Layr-Labs/sidecar/internal/clients/ethereum"
	"github.com/Layr-Labs/sidecar/internal/clients/etherscan"
	"github.com/Layr-Labs/sidecar/internal/config"
	"github.com/Layr-Labs/sidecar/internal/contractManager"
	"github.com/Layr-Labs/sidecar/internal/contractStore"
	"github.com/Layr-Labs/sidecar/internal/fetcher"
	"github.com/Layr-Labs/sidecar/internal/parser"
	"github.com/Layr-Labs/sidecar/internal/storage/metadata"
	"go.uber.org/zap"
	"slices"
)

type Indexer struct {
	Logger          *zap.Logger
	MetadataStore   metadata.MetadataStore
	ContractStore   contractStore.ContractStore
	ContractManager *contractManager.ContractManager
	EtherscanClient *etherscan.EtherscanClient
	Fetcher         *fetcher.Fetcher
	EthereumClient  *ethereum.Client
	Config          *config.Config
}

func NewIndexer(
	ms metadata.MetadataStore,
	cs contractStore.ContractStore,
	es *etherscan.EtherscanClient,
	cm *contractManager.ContractManager,
	e *ethereum.Client,
	f *fetcher.Fetcher,
	l *zap.Logger,
	cfg *config.Config,
) *Indexer {
	return &Indexer{
		Logger:          l,
		MetadataStore:   ms,
		ContractStore:   cs,
		EtherscanClient: es,
		ContractManager: cm,
		Fetcher:         f,
		EthereumClient:  e,
		Config:          cfg,
	}
}

func (idx *Indexer) FetchAndIndexBlock(ctx context.Context, blockNumber uint64, reindex bool) (*fetcher.FetchedBlock, *metadata.Block, bool, error) {
	previouslyIndexed := false
	b, err := idx.Fetcher.FetchBlock(ctx, blockNumber)
	if err != nil {
		idx.Logger.Sugar().Errorw(fmt.Sprintf("Failed to fetch block: %v", blockNumber), zap.Error(err))
		return b, nil, previouslyIndexed, err
	}

	block, err, alreadyIndexed := idx.IndexFetchedBlock(ctx, b)
	previouslyIndexed = alreadyIndexed
	if err != nil {
		idx.Logger.Sugar().Errorw(fmt.Sprintf("Failed to index block: %v", blockNumber), zap.Error(err))
		return b, nil, previouslyIndexed, err
	}
	// If we want to re-index the block, continue
	if alreadyIndexed && reindex == false {
		return b, block, previouslyIndexed, nil
	}

	idx.Logger.Sugar().Debugw("Indexing transactions", zap.Uint64("blockNumber", blockNumber))
	// If we're inserting for the first time, insert as a batch, otherwise process them one-by-one
	_, err = idx.IndexTransactions(ctx, block, b, reindex == false)
	if err != nil {
		idx.Logger.Sugar().Errorw(fmt.Sprintf("Failed to index transactions: %v", blockNumber), zap.Error(err))
		return b, nil, previouslyIndexed, err
	}
	return b, block, previouslyIndexed, nil
}

func (idx *Indexer) ParseAndIndexTransactionLogs(ctx context.Context, fetchedBlock *fetcher.FetchedBlock, blockSequenceId uint64) {
	for i, tx := range fetchedBlock.Block.Transactions {
		txReceipt, ok := fetchedBlock.TxReceipts[tx.Hash.Value()]
		if !ok {
			continue
		}

		parsedTransactionLogs, err := idx.ParseTransactionLogs(tx, txReceipt)
		if err != nil {
			idx.Logger.Sugar().Errorw("failed to process transaction logs",
				zap.Error(err),
				zap.String("txHash", tx.Hash.Value()),
				zap.Uint64("block", tx.BlockNumber.Value()),
			)
			continue
		}
		if parsedTransactionLogs == nil {
			idx.Logger.Sugar().Debugw("Log line is nil",
				zap.String("txHash", tx.Hash.Value()),
				zap.Uint64("block", tx.BlockNumber.Value()),
				zap.Int("logIndex", i),
			)
			continue
		}

		for _, log := range parsedTransactionLogs.Logs {
			_, err := idx.IndexLog(ctx, fetchedBlock.Block.Number.Value(), blockSequenceId, tx.Hash.Value(), tx.Index.Value(), log)
			if err != nil {
				idx.Logger.Sugar().Errorw("failed to index log",
					zap.Error(err),
					zap.String("txHash", tx.Hash.Value()),
					zap.Uint64("block", tx.BlockNumber.Value()),
				)
			}
		}

		upgradedLogs := idx.FindContractUpgradedLogs(parsedTransactionLogs.Logs)
		if len(upgradedLogs) > 0 {
			idx.Logger.Sugar().Debugw("Found contract upgrade logs",
				zap.String("txHash", tx.Hash.Value()),
				zap.Uint64("block", tx.BlockNumber.Value()),
				zap.Int("count", len(upgradedLogs)),
			)

			idx.IndexContractUpgrades(fetchedBlock.Block.Number.Value(), upgradedLogs, false)
		}
	}
}

func (idx *Indexer) IndexFetchedBlock(ctx context.Context, fetchedBlock *fetcher.FetchedBlock) (*metadata.Block, error, bool) {
	blockNumber := fetchedBlock.Block.Number.Value()
	blockHash := fetchedBlock.Block.Hash.Value()

	foundBlock, err := idx.MetadataStore.GetBlockByNumber(blockNumber)
	if err != nil {
		idx.Logger.Sugar().Errorw("Failed to get block by number",
			zap.Error(err),
			zap.Uint64("blockNumber", blockNumber),
		)
		return nil, err, false
	}
	if foundBlock != nil {
		idx.Logger.Sugar().Debugw(fmt.Sprintf("Block '%d' already indexed", blockNumber))
		return foundBlock, nil, true
	}

	insertedBlock, err := idx.MetadataStore.InsertBlockAtHeight(blockNumber, blockHash, fetchedBlock.Block.Timestamp.Value())
	if err != nil {
		idx.Logger.Sugar().Errorw("Failed to insert block at height",
			zap.Error(err),
			zap.Uint64("blockNumber", blockNumber),
			zap.String("blockHash", blockHash),
		)
		return nil, err, false
	}

	return insertedBlock, nil, false
}

func (idx *Indexer) isInterestingAddress(addr string) bool {
	return slices.Contains(idx.Config.GetInterestingAddressForConfigEnv(), addr)
}

func (idx *Indexer) IndexTransactions(
	ctx context.Context,
	block *metadata.Block,
	fetchedBlock *fetcher.FetchedBlock,
	asBatch bool,
) ([]*metadata.Transaction, error) {
	indexedTransactions := make([]*metadata.Transaction, 0)

	var err error
	txsToInsert := make([]metadata.BatchTransaction, 0)
	for _, tx := range fetchedBlock.Block.Transactions {

		insertTx := metadata.BatchTransaction{
			TxHash:  tx.Hash.Value(),
			TxIndex: tx.Index.Value(),
			From:    tx.From.Value(),
			To:      tx.To.Value(),
		}
		txReceipt, ok := fetchedBlock.TxReceipts[tx.Hash.Value()]
		if ok {
			insertTx.ContractAddress = txReceipt.ContractAddress.Value()
			insertTx.BytecodeHash = txReceipt.GetBytecodeHash()
		}

		hasInterestingLog := false
		if ok {
			for _, log := range txReceipt.Logs {
				if idx.isInterestingAddress(log.Address.Value()) {
					hasInterestingLog = true
				}
			}
		}

		// Only insert transactions that are interesting:
		// - TX is being sent to an EL contract
		// - TX created an EL contract
		// - TX has logs emitted by an EL contract
		if hasInterestingLog || idx.isInterestingAddress(tx.To.Value()) || (ok && idx.isInterestingAddress(txReceipt.ContractAddress.Value())) {
			txsToInsert = append(txsToInsert, insertTx)
		}
	}
	if asBatch {
		indexedTransactions, err = idx.MetadataStore.BatchInsertBlockTransactions(block.Id, block.Number, txsToInsert)
	} else {
		for _, tx := range txsToInsert {
			_, err := idx.MetadataStore.InsertBlockTransaction(block.Id, block.Number, tx.TxHash, tx.TxIndex, tx.From, tx.To, tx.ContractAddress, tx.BytecodeHash)
			if err != nil {
				idx.Logger.Sugar().Errorw("Failed to insert block transaction",
					zap.Error(err),
					zap.String("txHash", tx.TxHash),
					zap.Uint64("blockNumber", block.Number),
				)
			} else {
				idx.Logger.Sugar().Debugw("Inserted block transaction", zap.String("txHash", tx.TxHash), zap.Uint64("blockNumber", block.Number))
			}
		}
	}
	if err != nil {
		idx.Logger.Sugar().Errorw("Failed to batch insert block transactions",
			zap.Error(err),
			zap.Uint64("blockNumber", block.Number),
		)
		return nil, err
	}
	// More performant to loop twice and use a bulk insert than to insert one by one
	for _, tx := range fetchedBlock.Block.Transactions {
		txReceipt, ok := fetchedBlock.TxReceipts[tx.Hash.Value()]
		if !ok {
			continue
		}

		idx.Logger.Sugar().Debug("processing transaction", zap.String("txHash", tx.Hash.Value()))
		contractAddress := txReceipt.ContractAddress.Value()
		eip1197StoredValue := ""
		if contractAddress != "" {
			eip1197StoredValue = fetchedBlock.ContractStorage[contractAddress]
		}

		if txReceipt.ContractAddress.Value() != "" {
			idx.handleContractCreation(
				txReceipt.ContractAddress.Value(),
				txReceipt.GetBytecodeHash(),
				eip1197StoredValue,
				block.Number,
				false,
			)
		}
	}

	return indexedTransactions, nil
}

// Handles indexing a contract created by a transaction
// Does NOT include contracts that are part of logs
func (idx *Indexer) IndexContractsForBlock(
	block *metadata.Block,
	fetchedBlock *fetcher.FetchedBlock,
	reindexContract bool,
) {
	for _, tx := range fetchedBlock.Block.Transactions {
		txReceipt, ok := fetchedBlock.TxReceipts[tx.Hash.Value()]
		if !ok {
			continue
		}

		idx.Logger.Sugar().Debug("processing transaction", zap.String("txHash", tx.Hash.Value()))
		contractAddress := txReceipt.ContractAddress.Value()
		eip1967StoredValue := ""
		if contractAddress != "" {
			eip1967StoredValue = fetchedBlock.ContractStorage[contractAddress]
		}

		// If the transaction created a contract, index it
		if txReceipt.ContractAddress.Value() != "" {
			idx.handleContractCreation(
				txReceipt.ContractAddress.Value(),
				txReceipt.GetBytecodeHash(),
				eip1967StoredValue,
				block.Number,
				reindexContract,
			)
		}
	}
}

func (idx *Indexer) handleContractCreation(
	contractAddress string,
	bytecodeHash string,
	eip1197StoredValue string,
	blockNumber uint64,
	reindexContract bool,
) {
	_, err := idx.ContractManager.CreateContract(contractAddress, bytecodeHash, reindexContract)
	if err != nil {
		idx.Logger.Sugar().Errorw("Failed to get find or create address",
			zap.Error(err),
			zap.String("contractAddress", contractAddress),
		)
	}
	if len(eip1197StoredValue) == 66 {
		idx.ContractManager.HandleProxyContractCreation(contractAddress, eip1197StoredValue, blockNumber, reindexContract)
	}
}

func (idx *Indexer) IndexLog(
	ctx context.Context,
	blockNumber uint64,
	blockSequenceId uint64,
	txHash string,
	txIndex uint64,
	log *parser.DecodedLog,
) (*metadata.TransactionLog, error) {
	insertedLog, err := idx.MetadataStore.InsertTransactionLog(
		txHash,
		txIndex,
		blockNumber,
		blockSequenceId,
		log,
		log.OutputData,
	)
	if err != nil {
		idx.Logger.Sugar().Errorw("Failed to insert transaction log",
			zap.Error(err),
			zap.String("txHash", txHash),
			zap.Uint64("blockNumber", blockNumber),
		)
		return nil, err
	}

	return insertedLog, nil
}
