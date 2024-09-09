package sqlite

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Layr-Labs/go-sidecar/internal/config"
	"github.com/Layr-Labs/go-sidecar/internal/parser"
	"github.com/Layr-Labs/go-sidecar/internal/storage"
	"go.uber.org/zap"
	"golang.org/x/xerrors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"strings"
	"time"
)

type SqliteBlockStoreConfig struct {
	DbLocation string
}

type SqliteBlockStore struct {
	Db           *gorm.DB
	migrated     bool
	Logger       *zap.Logger
	GlobalConfig *config.Config
}

func NewSqliteBlockStore(db *gorm.DB, l *zap.Logger, cfg *config.Config) *SqliteBlockStore {
	bs := &SqliteBlockStore{
		Db:           db,
		Logger:       l,
		GlobalConfig: cfg,
	}
	return bs
}

func (s *SqliteBlockStore) InsertBlockAtHeight(
	blockNumber uint64,
	hash string,
	blockTime uint64,
) (*storage.Block, error) {
	block := &storage.Block{
		Number:    blockNumber,
		Hash:      hash,
		BlockTime: time.Unix(int64(blockTime), 0),
	}

	res := s.Db.Model(&storage.Block{}).Clauses(clause.Returning{}).Create(&block)

	if res.Error != nil {
		return nil, fmt.Errorf("failed to insert block with number '%d': %w", blockNumber, res.Error)
	}
	return block, nil
}

func (s *SqliteBlockStore) InsertBlockTransaction(
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

	tx := &storage.Transaction{
		BlockNumber:      blockNumber,
		TransactionHash:  txHash,
		TransactionIndex: txIndex,
		FromAddress:      from,
		ToAddress:        to,
		ContractAddress:  contractAddress,
		BytecodeHash:     bytecodeHash,
	}

	result := s.Db.Model(&storage.Transaction{}).Clauses(clause.Returning{}).Create(&tx)

	if result.Error != nil {
		return nil, xerrors.Errorf("Failed to insert block transaction '%d' - '%s': %w", blockNumber, txHash, result.Error)
	}
	return tx, nil
}

func (s *SqliteBlockStore) InsertTransactionLog(
	txHash string,
	transactionIndex uint64,
	blockNumber uint64,
	log *parser.DecodedLog,
	outputData map[string]interface{},
) (*storage.TransactionLog, error) {
	argsJson, err := json.Marshal(log.Arguments)
	if err != nil {
		s.Logger.Sugar().Errorw("Failed to marshal arguments", zap.Error(err))
	}

	outputDataJson := []byte{}
	outputDataJson, err = json.Marshal(outputData)
	if err != nil {
		s.Logger.Sugar().Errorw("Failed to marshal output data", zap.Error(err))
	}

	txLog := &storage.TransactionLog{
		TransactionHash:  txHash,
		TransactionIndex: transactionIndex,
		BlockNumber:      blockNumber,
		Address:          strings.ToLower(log.Address),
		Arguments:        string(argsJson),
		EventName:        log.EventName,
		LogIndex:         log.LogIndex,
		OutputData:       string(outputDataJson),
	}
	result := s.Db.Model(&storage.TransactionLog{}).Clauses(clause.Returning{}).Create(&txLog)

	if result.Error != nil {
		return nil, xerrors.Errorf("Failed to insert transaction log: %w - %+v", result.Error, txLog)
	}
	return txLog, nil
}

func (s *SqliteBlockStore) GetLatestBlock() (*storage.Block, error) {
	block := &storage.Block{}

	query := `
	select
	 *
	from blocks
	order by number desc
	limit 1`

	result := s.Db.Model(&storage.Block{}).Raw(query).Scan(&block)
	if result.Error != nil {
		return nil, xerrors.Errorf("Failed to get latest block: %w", result.Error)
	}
	return block, nil
}

func (s *SqliteBlockStore) GetBlockByNumber(blockNumber uint64) (*storage.Block, error) {
	block := &storage.Block{}

	result := s.Db.Model(block).Where("number = ?", blockNumber).First(&block)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, result.Error
	}
	return block, nil
}

func (s *SqliteBlockStore) InsertOperatorRestakedStrategies(
	avsDirectorAddress string,
	blockNumber uint64,
	blockTime time.Time,
	operator string,
	avs string,
	strategy string,
) (*storage.OperatorRestakedStrategies, error) {
	ors := &storage.OperatorRestakedStrategies{
		AvsDirectoryAddress: strings.ToLower(avsDirectorAddress),
		BlockNumber:         blockNumber,
		Operator:            operator,
		Avs:                 avs,
		Strategy:            strategy,
		BlockTime:           blockTime,
	}

	result := s.Db.Model(&storage.OperatorRestakedStrategies{}).Clauses(clause.Returning{}).Create(&ors)

	if result.Error != nil {
		return nil, xerrors.Errorf("Failed to insert operator restaked strategies: %w", result.Error)
	}
	return ors, nil
}

func (s *SqliteBlockStore) GetLatestActiveAvsOperators(blockNumber uint64, avsDirectoryAddress string) ([]*storage.ActiveAvsOperator, error) {
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
	result := s.Db.Raw(query, avsDirectoryAddress, blockNumber).Scan(&rows)
	if result.Error != nil {
		return nil, xerrors.Errorf("Failed to get latest active AVS operators: %w", result.Error)
	}
	return rows, nil
}
