package sqlite

import (
	"github.com/Layr-Labs/sidecar/internal/config"
	"go.uber.org/zap"
	"gorm.io/gorm"
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

/*
func (s *SqliteBlockStore) GetNextSequenceId() (uint64, error) {

}
func (s *SqliteBlockStore) InsertBlockAtHeight(blockNumber uint64, hash string, blockTime uint64) (*storage.Block, error) {

}
func (s *SqliteBlockStore) UpdateBlockPath(sequenceId uint64, blockNumber uint64, path string) (*storage.Block, error) {

}
func (s *SqliteBlockStore) InsertBlockTransaction(sequenceId uint64, blockNumber uint64, txHash string, txIndex uint64, from string, to string, contractAddress string, bytecodeHash string) (*storage.Transaction, error) {

}
func (s *SqliteBlockStore) InsertTransactionLog(txHash string, transactionIndex uint64, blockNumber uint64, blockSequenceId uint64, log *parser.DecodedLog, outputData map[string]interface{}) (*storage.Transaction, error) {

}
func (s *SqliteBlockStore) BatchInsertBlockTransactions(sequenceId uint64, blockNumber uint64, transactions []storage.BatchTransaction) ([]*storage.Transaction, error) {

}
func (s *SqliteBlockStore) BatchInsertTransactionLogs(transactions []*BatchInsertTransactionLogs) ([]*TransactionLog, error) {

}
func (s *SqliteBlockStore) GetLatestBlock() (int64, error) {

}
func (s *SqliteBlockStore) GetBlockByNumber(blockNumber uint64) (*Block, error) {

}
func (s *SqliteBlockStore) DeleteTransactionLogsForBlock(blockNumber uint64) error {

}
func (s *SqliteBlockStore) GetTransactionByHash(txHash string) (*Transaction, error) {

}
func (s *SqliteBlockStore) ListTransactionsForContractAddress(contractAddress string) ([]*Transaction, error) {

}
func (s *SqliteBlockStore) ListTransactionLogsForContractAddress(contractAddress string) ([]*TransactionLog, error) {

}
func (s *SqliteBlockStore) InsertOperatorRestakedStrategies(avsDirectorAddress string, blockNumber uint64, blockTime time.Time, operator string, avs string, strategy string) (*OperatorRestakedStrategies, error) {

}*/
