package storage

import (
	"github.com/Layr-Labs/sidecar/internal/parser"
	"time"
)

type BlockStore interface {
	InsertBlockAtHeight(blockNumber uint64, hash string, blockTime uint64) (*Block, error)
	InsertBlockTransaction(blockNumber uint64, txHash string, txIndex uint64, from string, to string, contractAddress string, bytecodeHash string) (*Transaction, error)
	InsertTransactionLog(txHash string, transactionIndex uint64, blockNumber uint64, log *parser.DecodedLog, outputData map[string]interface{}) (*TransactionLog, error)
	GetLatestBlock() (*Block, error)
	GetBlockByNumber(blockNumber uint64) (*Block, error)
	InsertOperatorRestakedStrategies(avsDirectorAddress string, blockNumber uint64, blockTime time.Time, operator string, avs string, strategy string) (*OperatorRestakedStrategies, error)

	// Less generic functions
	GetLatestActiveAvsOperators(blockNumber uint64, avsDirectoryAddress string) ([]*ActiveAvsOperator, error)
}

// Tables
type Block struct {
	Number    uint64
	Hash      string
	BlockTime time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt time.Time
}

type Transaction struct {
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

type ActiveAvsOperator struct {
	Avs      string
	Operator string
}
