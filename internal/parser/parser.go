package parser

import "github.com/Layr-Labs/go-sidecar/internal/clients/ethereum"

type ParsedTransaction struct {
	MethodName  string
	DecodedData string
	Logs        []*DecodedLog
	Transaction *ethereum.EthereumTransaction
	Receipt     *ethereum.EthereumTransactionReceipt
}

type DecodedLog struct {
	LogIndex   uint64
	Address    string
	Arguments  []Argument
	EventName  string
	OutputData map[string]interface{}
}

type Argument struct {
	Name    string
	Type    string
	Value   interface{}
	Indexed bool
}
