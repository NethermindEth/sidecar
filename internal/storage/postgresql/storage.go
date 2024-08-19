package postgresql

import (
	"encoding/json"
	"errors"
	"github.com/Layr-Labs/sidecar/internal/parser"
	pg "github.com/Layr-Labs/sidecar/internal/postgres"
	"github.com/Layr-Labs/sidecar/internal/storage"
	"go.uber.org/zap"
	"golang.org/x/xerrors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"strings"
	"time"
)

type PostgresBlockStoreConfig struct {
	DBHost     string
	DBPort     int
	DBUsername string
	DBPassword string
	DBName     string
}

type PostgresBlockStore struct {
	Db       *gorm.DB
	migrated bool
	Logger   *zap.Logger
}

func NewPostgresBlockStore(db *gorm.DB, l *zap.Logger) (*PostgresBlockStore, error) {
	mds := &PostgresBlockStore{
		Db:     db,
		Logger: l,
	}

	mds.autoMigrate()

	return mds, nil
}

func (m *PostgresBlockStore) autoMigrate() {
	if m.migrated {
		return
	}

	m.migrated = true
}

func (m *PostgresBlockStore) CreateTxBlock(tx *gorm.DB) *gorm.DB {
	if tx != nil {
		return tx
	}
	return m.Db.Begin()
}

func (m *PostgresBlockStore) GetNextSequenceId() (uint64, error) {
	return pg.WrapTxAndCommit[uint64](func(txn *gorm.DB) (uint64, error) {
		query := `SELECT coalesce(max(id), -1) + 1 FROM block_sequences`

		var nextId uint64
		result := txn.Raw(query).Scan(&nextId)

		if result.Error != nil {
			return 0, xerrors.Errorf("Failed to get next sequence id: %w", result.Error)
		}

		return nextId, nil
	}, nil, m.Db)
}

func (m *PostgresBlockStore) UpdateBlockPath(sequenceId uint64, blockNumber uint64, path string) (*storage.Block, error) {
	return pg.WrapTxAndCommit[*storage.Block](func(txn *gorm.DB) (*storage.Block, error) {
		sequence := &storage.Block{}
		result := txn.Model(sequence).
			Clauses(clause.Returning{}).
			Where("id = ? and number = ?", sequenceId, blockNumber).
			Update("blob_path", path)

		if result.Error != nil {
			return nil, xerrors.Errorf("Failed to update block path: %w", result.Error)
		}
		return sequence, nil
	}, nil, m.Db)
}

func (m *PostgresBlockStore) InsertBlockAtHeight(
	blockNumber uint64,
	hash string,
	blockTime uint64,
) (*storage.Block, error) {
	return pg.WrapTxAndCommit[*storage.Block](func(txn *gorm.DB) (*storage.Block, error) {

		blockSeq := &storage.Block{
			Number:    blockNumber,
			Hash:      hash,
			BlockTime: time.Unix(int64(blockTime), 0),
		}

		result := m.Db.Model(&storage.Block{}).Clauses(clause.Returning{}).Create(&blockSeq)

		if result.Error != nil {
			return nil, xerrors.Errorf("Failed to insert block sequence: %w", result.Error)
		}
		return blockSeq, nil
	}, nil, m.Db)
}

func (m *PostgresBlockStore) InsertBlockTransaction(
	sequenceId uint64,
	blockNumber uint64,
	txHash string,
	txIndex uint64,
	from string,
	to string,
	contractAddress string,
	bytecodeHash string,
) (*storage.Transaction, error) {
	to = strings.ToLower(to)
	from = strings.ToLower(from)
	contractAddress = strings.ToLower(contractAddress)
	return pg.WrapTxAndCommit[*storage.Transaction](func(txn *gorm.DB) (*storage.Transaction, error) {
		tx := &storage.Transaction{
			BlockSequenceId:  sequenceId,
			BlockNumber:      blockNumber,
			TransactionHash:  txHash,
			TransactionIndex: txIndex,
			FromAddress:      from,
			ToAddress:        to,
			ContractAddress:  contractAddress,
			BytecodeHash:     bytecodeHash,
		}

		result := m.Db.Model(&storage.Transaction{}).Clauses(clause.Returning{}).Create(&tx)

		if result.Error != nil {
			return nil, xerrors.Errorf("Failed to insert block transaction: %w", result.Error)
		}
		return tx, nil
	}, nil, m.Db)
}

func (m *PostgresBlockStore) BatchInsertBlockTransactions(
	sequenceId uint64,
	blockNumber uint64,
	transactions []storage.BatchTransaction,
) ([]*storage.Transaction, error) {
	if len(transactions) == 0 {
		return make([]*storage.Transaction, 0), nil
	}
	return pg.WrapTxAndCommit[[]*storage.Transaction](func(txn *gorm.DB) ([]*storage.Transaction, error) {
		txs := make([]*storage.Transaction, 0, len(transactions))
		for _, tx := range transactions {
			txs = append(txs, &storage.Transaction{
				BlockSequenceId:  sequenceId,
				BlockNumber:      blockNumber,
				TransactionHash:  tx.TxHash,
				TransactionIndex: tx.TxIndex,
				FromAddress:      tx.From,
				ToAddress:        tx.To,
				ContractAddress:  tx.ContractAddress,
				BytecodeHash:     tx.BytecodeHash,
			})
		}

		result := m.Db.Model(&storage.Transaction{}).Clauses(clause.Returning{}).Create(&txs)

		if result.Error != nil {
			return nil, xerrors.Errorf("Failed to insert block transaction: %w", result.Error)
		}
		return txs, nil
	}, nil, m.Db)
}

func (m *PostgresBlockStore) InsertTransactionLog(
	txHash string,
	transactionIndex uint64,
	blockNumber uint64,
	blockSequenceId uint64,
	log *parser.DecodedLog,
	outputData map[string]interface{},
) (*storage.TransactionLog, error) {
	return pg.WrapTxAndCommit[*storage.TransactionLog](func(txn *gorm.DB) (*storage.TransactionLog, error) {
		argsJson, err := json.Marshal(log.Arguments)
		if err != nil {
			m.Logger.Sugar().Errorw("Failed to marshal arguments", zap.Error(err))
		}

		outputDataJson := []byte{}
		outputDataJson, err = json.Marshal(outputData)
		if err != nil {
			m.Logger.Sugar().Errorw("Failed to marshal output data", zap.Error(err))
		}

		txLog := &storage.TransactionLog{
			TransactionHash:  txHash,
			TransactionIndex: transactionIndex,
			BlockNumber:      blockNumber,
			BlockSequenceId:  blockSequenceId,
			Address:          strings.ToLower(log.Address),
			Arguments:        string(argsJson),
			EventName:        log.EventName,
			LogIndex:         log.LogIndex,
			OutputData:       string(outputDataJson),
		}
		result := m.Db.Model(&storage.TransactionLog{}).Clauses(clause.Returning{}).Create(&txLog)

		if result.Error != nil {
			return nil, xerrors.Errorf("Failed to insert transaction log: %w - %+v", result.Error, txLog)
		}
		return txLog, nil
	}, nil, m.Db)
}

func (m *PostgresBlockStore) BatchInsertTransactionLogs(transactions []*storage.BatchInsertTransactionLogs) ([]*storage.TransactionLog, error) {
	logs := make([]*storage.TransactionLog, 0)

	for _, tx := range transactions {
		for _, log := range tx.ParsedTransaction.Logs {
			argsJson, err := json.Marshal(log.Arguments)
			if err != nil {
				m.Logger.Sugar().Errorw("Failed to marshal arguments", zap.Error(err))
			}
			logs = append(logs, &storage.TransactionLog{
				TransactionHash:  tx.Transaction.Hash.Value(),
				TransactionIndex: tx.Transaction.Index.Value(),
				BlockNumber:      tx.Transaction.BlockNumber.Value(),
				BlockSequenceId:  0,
				Address:          strings.ToLower(log.Address),
				Arguments:        string(argsJson),
				EventName:        log.EventName,
				LogIndex:         log.LogIndex,
			})
		}
	}

	result := m.Db.Model(&storage.TransactionLog{}).Clauses(clause.Returning{}).Create(&logs)

	if result.Error != nil {
		return nil, xerrors.Errorf("Failed to insert block transaction: %w", result.Error)
	}
	return logs, nil
}

type latestBlockNumber struct {
	BlockNumber uint64
}

func (m *PostgresBlockStore) GetLatestBlock() (int64, error) {
	block := &latestBlockNumber{}

	query := `select coalesce(max(number), 0) as block_number from blocks`

	result := m.Db.Raw(query).Scan(&block)
	if result.Error != nil {
		return 0, xerrors.Errorf("Failed to get latest block: %w", result.Error)
	}
	return int64(block.BlockNumber), nil
}

func (m *PostgresBlockStore) GetBlockByNumber(blockNumber uint64) (*storage.Block, error) {
	block := &storage.Block{}

	result := m.Db.Model(block).Where("number = ?", blockNumber).First(&block)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return block, nil
}

func (m *PostgresBlockStore) DeleteTransactionLogsForBlock(blockNumber uint64) error {
	result := m.Db.Where("block_number = ?", blockNumber).Delete(&storage.TransactionLog{})
	if result.Error != nil {
		return xerrors.Errorf("Failed to delete transaction logs: %w", result.Error)
	}
	return nil
}

func (m *PostgresBlockStore) GetTransactionByHash(txHash string) (*storage.Transaction, error) {
	tx := &storage.Transaction{}

	result := m.Db.Model(&tx).Where("transaction_hash = ?", strings.ToLower(txHash)).First(&tx)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return tx, nil
}

func (m *PostgresBlockStore) ListTransactionsForContractAddress(contractAddress string) ([]*storage.Transaction, error) {
	contractAddress = strings.ToLower(contractAddress)
	var txs []*storage.Transaction

	result := m.Db.Model(&storage.Transaction{}).
		Where("contract_address = ? or to_address = ?", contractAddress, contractAddress).
		Find(&txs)
	if result.Error != nil {
		return nil, xerrors.Errorf("Failed to list transactions for contract address: %w", result.Error)
	}
	return txs, nil
}

func (m *PostgresBlockStore) ListTransactionLogsForContractAddress(contractAddress string) ([]*storage.TransactionLog, error) {
	contractAddress = strings.ToLower(contractAddress)
	var txLogs []*storage.TransactionLog

	result := m.Db.Model(&storage.TransactionLog{}).
		Where("address = ?", contractAddress).
		Find(&txLogs)
	if result.Error != nil {
		return nil, xerrors.Errorf("Failed to list transaction logs for contract address: %w", result.Error)
	}
	return txLogs, nil
}

func (m *PostgresBlockStore) InsertOperatorRestakedStrategies(avsDirectorAddress string, blockNumber uint64, blockTime time.Time, operator string, avs string, strategy string) (*storage.OperatorRestakedStrategies, error) {
	return pg.WrapTxAndCommit[*storage.OperatorRestakedStrategies](func(txn *gorm.DB) (*storage.OperatorRestakedStrategies, error) {
		ors := &storage.OperatorRestakedStrategies{
			AvsDirectoryAddress: strings.ToLower(avsDirectorAddress),
			BlockNumber:         blockNumber,
			Operator:            operator,
			Avs:                 avs,
			Strategy:            strategy,
			BlockTime:           blockTime,
		}

		result := m.Db.Model(&storage.OperatorRestakedStrategies{}).Clauses(clause.Returning{}).Create(&ors)

		if result.Error != nil {
			return nil, xerrors.Errorf("Failed to insert operator restaked strategies: %w", result.Error)
		}
		return ors, nil
	}, nil, m.Db)

}

func (m *PostgresBlockStore) GetLatestActiveAvsOperators(blockNumber uint64, avsDirectoryAddress string) ([]*storage.ActiveAvsOperator, error) {
	avsDirectoryAddress = strings.ToLower(avsDirectoryAddress)

	rows := make([]*storage.ActiveAvsOperator, 0)
	query := `
		WITH latest_status AS (
			SELECT 
				lower(tl.arguments #>> '{0,Value}') as operator,
				lower(tl.arguments #>> '{1,Value}') as avs,
				lower(tl.output_data #>> '{status}') as status,
				ROW_NUMBER() OVER (PARTITION BY lower(tl.arguments #>> '{0,Value}'), lower(tl.arguments #>> '{1,Value}') ORDER BY block_number DESC) AS row_number
			FROM transaction_logs as tl
			WHERE
				tl.address = ?
				AND tl.event_name = 'OperatorAVSRegistrationStatusUpdated'
				AND tl.block_number <= ?
		)
		SELECT avs, operator
		FROM latest_status
		WHERE row_number = 1 AND status = '1';
	`
	result := m.Db.Raw(query, avsDirectoryAddress, blockNumber).Scan(&rows)
	if result.Error != nil {
		return nil, xerrors.Errorf("Failed to get latest active AVS operators: %w", result.Error)
	}
	return rows, nil
}
