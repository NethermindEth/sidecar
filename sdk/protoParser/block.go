package protoParser

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/Layr-Labs/sidecar/protos/eigenlayer/blocklake/v1"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

type Parser struct {
	Client *ethclient.Client
}

func NewParser(c *ethclient.Client) *Parser {
	return &Parser{
		Client: c,
	}
}

type TransactionWithHash struct {
	Transaction *types.Transaction
	Receipt     *types.Receipt
	TxHash      common.Hash
}

func (p *Parser) FetchTransactionReceipts(transactionHashes []common.Hash) ([]*types.Receipt, error) {
	var receipts = []*types.Receipt{}
	for _, hash := range transactionHashes {
		r, err := p.Client.TransactionReceipt(context.Background(), hash)
		if err != nil {
			fmt.Printf("Failed to get receipt for hash '%s'\n", hash.String())
		}
		receipts = append(receipts, r)
	}
	return receipts, nil
}

func (p *Parser) ParseTransactionReceiptToProto(receipt *types.Receipt) (*v1.TransactionReceipt, error) {
	return &v1.TransactionReceipt{
		TransactionHash:   receipt.TxHash.String(),
		TransactionIndex:  uint64(receipt.TransactionIndex),
		BlockHash:         receipt.BlockHash.String(),
		BlockNumber:       receipt.BlockNumber.Uint64(),
		To:                "",
		From:              "",
		GasUsed:           0,
		CumulativeGasUsed: 0,
		ContractAddress:   "",
		LogsBloom:         "",
		Type:              0,
		EffectiveGasPrice: 0,
		Status:            0,
	}, nil
}

func (p *Parser) ParseTransactionsToProto(transactions []*types.Transaction) ([]*v1.Transaction, error) {
	var parsedTransactions = []*v1.Transaction{}

	txHashes := make([]common.Hash, 0)

	for _, t := range transactions {
		txHashes = append(txHashes, t.Hash())
	}

	receipts, err := p.FetchTransactionReceipts(txHashes)
	if err != nil {
		fmt.Printf("Error fetching receipts: %+v\n", err)
	}
	receiptsMap := make(map[common.Hash]*types.Receipt)

	for _, r := range receipts {
		receiptsMap[r.TxHash] = r
	}

	for i, t := range transactions {
		v, r, s := t.RawSignatureValues()

		jsonBlock, _ := json.MarshalIndent(receiptsMap[t.Hash()], "", "\t")

		fmt.Printf("Receipt: %+v\n", string(jsonBlock))

		parsedTransaction := &v1.Transaction{
			Hash:     t.Hash().String(),
			Size:     t.Size(),
			From:     "",
			To:       t.To().String(),
			Gas:      t.Gas(),
			GasPrice: t.GasPrice().Uint64(),
			Input:    "",
			Nonce:    t.Nonce(),
			Index:    uint64(i),
			Value:    t.Value().String(),
			Type:     uint64(t.Type()),
			SignatureValues: &v1.Transaction_SignatureValues{
				V: v.String(),
				R: r.String(),
				S: s.String(),
			},
		}
		parsedTransactions = append(parsedTransactions, parsedTransaction)
	}
	return parsedTransactions, nil
}

func (p *Parser) ParseWithdrawalsToProto(withdrawals []*types.Withdrawal) ([]*v1.Withdrawal, error) {
	var parsedWithdrawals = []*v1.Withdrawal{}

	for _, w := range withdrawals {
		parsedWithdrawal := &v1.Withdrawal{
			Index:     w.Index,
			Validator: w.Validator,
			Address:   w.Address.String(),
			Amount:    w.Amount,
		}
		parsedWithdrawals = append(parsedWithdrawals, parsedWithdrawal)
	}
	return parsedWithdrawals, nil
}

func (p *Parser) ParseBlockToProto(block *types.Block) (*v1.Block, error) {
	parsedBlock := &v1.Block{
		Hash: block.Hash().String(),
		Size: block.Size(),
		Header: &v1.BlockHeader{
			ParentHash:            block.ParentHash().String(),
			Sha3Uncles:            block.Header().UncleHash.String(),
			Miner:                 block.Header().Coinbase.String(),
			StateRoot:             block.Header().Root.String(),
			TransactionsRoot:      block.Header().TxHash.String(),
			ReceiptsRoot:          block.Header().ReceiptHash.String(),
			LogsBloom:             block.Header().Bloom.Bytes(),
			Difficulty:            block.Header().Difficulty.Bytes(),
			Number:                block.Header().Number.String(),
			GasLimit:              block.Header().GasLimit,
			GasUsed:               block.Header().GasUsed,
			Timestamp:             block.Header().Time,
			ExtraData:             block.Header().Extra,
			MixHash:               block.Header().MixDigest.String(),
			Nonce:                 block.Header().Nonce.Uint64(),
			BaseFeePerGas:         block.Header().BaseFee.Bytes(),
			WithdrawalsRoot:       block.Header().WithdrawalsHash.String(),
			BlobGasUsed:           *block.BlobGasUsed(),
			ExcessBlobGas:         *block.ExcessBlobGas(),
			ParentBeaconBlockRoot: block.Header().ParentBeaconRoot.String(),
		},
		Transactions: nil,
		Withdrawals:  nil,
		Uncles:       nil,
	}

	parsedTransactions, err := p.ParseTransactionsToProto(block.Transactions())
	if err != nil {
		return nil, fmt.Errorf("Failed to parse transactions - %+v", err)
	}
	parsedBlock.Transactions = parsedTransactions

	parsedWithdrawals, err := p.ParseWithdrawalsToProto(block.Withdrawals())
	if err != nil {
		return nil, fmt.Errorf("Failed to parse withdrawals - %+v", err)
	}
	parsedBlock.Withdrawals = parsedWithdrawals

	return parsedBlock, nil
}
