package storage

import (
	"github.com/Layr-Labs/sidecar/internal/clients/ethereum"
	"github.com/Layr-Labs/sidecar/internal/parser"
	"gorm.io/gorm"
	"time"
)

type BlockStore interface {
	GetNextSequenceId() (uint64, error)
	InsertBlockAtHeight(blockNumber uint64, hash string, blockTime uint64) (*Block, error)
	UpdateBlockPath(sequenceId uint64, blockNumber uint64, path string) (*Block, error)
	InsertBlockTransaction(sequenceId uint64, blockNumber uint64, txHash string, txIndex uint64, from string, to string, contractAddress string, bytecodeHash string) (*Transaction, error)
	InsertTransactionLog(txHash string, transactionIndex uint64, blockNumber uint64, blockSequenceId uint64, log *parser.DecodedLog, outputData map[string]interface{}) (*TransactionLog, error)
	BatchInsertBlockTransactions(sequenceId uint64, blockNumber uint64, transactions []BatchTransaction) ([]*Transaction, error)
	BatchInsertTransactionLogs(transactions []*BatchInsertTransactionLogs) ([]*TransactionLog, error)
	GetLatestBlock() (int64, error)
	GetBlockByNumber(blockNumber uint64) (*Block, error)
	DeleteTransactionLogsForBlock(blockNumber uint64) error
	GetTransactionByHash(txHash string) (*Transaction, error)
	ListTransactionsForContractAddress(contractAddress string) ([]*Transaction, error)
	ListTransactionLogsForContractAddress(contractAddress string) ([]*TransactionLog, error)
	InsertOperatorRestakedStrategies(avsDirectorAddress string, blockNumber uint64, blockTime time.Time, operator string, avs string, strategy string) (*OperatorRestakedStrategies, error)

	GetDb() *gorm.DB

	// Less generic functions
	GetLatestActiveAvsOperators(blockNumber uint64, avsDirectoryAddress string) ([]*ActiveAvsOperator, error)

	// State change functions
	// InsertIntoAvsOperatorChangesForBlock(blockNumber uint64) error
	// InsertIntoOperatorShareChangesForBlock(blockNumber uint64) error
	// InsertIntoStakerShareChangesForBlock(blockNumber uint64) error
	// InsertIntoStakerDelegationChangesForBlock(blockNumber uint64) error
	// InsertIntoActiveRewardSubmissionsForBlock(blockNumber uint64) error
	//
	// // Aggregate table functions
	// CloneRegisteredAvsOperatorsForNewBlock(newBlockNumber uint64) error
	// CloneOperatorSharesForNewBlock(newBlockNumber uint64) error
	// CloneStakerSharesForNewBlock(newBlockNumber uint64) error
	// CloneDelegatedStakersForNewBlock(newBlockNumber uint64) error
	// SetActiveRewardsForNewBlock(newBlockNumber uint64) error
	// SetActiveRewardForAllForNewBlock(newBlockNumber uint64) error
}

// Tables
type Block struct {
	Id        uint64 `gorm:"type:serial"`
	Number    uint64
	Hash      string
	BlockTime time.Time
	BlobPath  string
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt time.Time
}

type Transaction struct {
	BlockSequenceId  uint64
	BlockNumber      uint64
	TransactionHash  string
	TransactionIndex uint64
	FromAddress      string
	ToAddress        string
	ContractAddress  string
	BytecodeHash     string
	CreatedAt        time.Time
	UpdatedAt        time.Time
	DeletedAt        time.Time
}

type TransactionLog struct {
	TransactionHash  string
	TransactionIndex uint64
	BlockNumber      uint64
	BlockSequenceId  uint64
	Address          string
	Arguments        string
	EventName        string
	LogIndex         uint64
	OutputData       string
	CreatedAt        time.Time
	UpdatedAt        time.Time
	DeletedAt        time.Time
}

type BatchTransaction struct {
	TxHash          string
	TxIndex         uint64
	From            string
	To              string
	ContractAddress string
	BytecodeHash    string
}

type OperatorRestakedStrategies struct {
	Id                  uint64 `gorm:"type:serial"`
	AvsDirectoryAddress string
	BlockNumber         uint64
	Operator            string
	Avs                 string
	Strategy            string
	BlockTime           time.Time
	CreatedAt           time.Time
	UpdatedAt           time.Time
	DeletedAt           time.Time
}

// Not tables
type BatchInsertTransactionLogs struct {
	Transaction       *ethereum.EthereumTransaction
	ParsedTransaction *parser.ParsedTransaction
}

type ActiveAvsOperator struct {
	Avs      string
	Operator string
}
