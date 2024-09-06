package postgresql

import (
	"encoding/json"
	"errors"
	"github.com/Layr-Labs/sidecar/internal/config"
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
	Db           *gorm.DB
	migrated     bool
	Logger       *zap.Logger
	GlobalConfig *config.Config
}

func NewPostgresBlockStore(db *gorm.DB, cfg *config.Config, l *zap.Logger) (*PostgresBlockStore, error) {
	mds := &PostgresBlockStore{
		Db:           db,
		Logger:       l,
		GlobalConfig: cfg,
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
