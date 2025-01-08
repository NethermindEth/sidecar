package rpcServer

import (
	"context"
	"fmt"
	v1EventTypes "github.com/Layr-Labs/protocol-apis/gen/protos/eigenlayer/sidecar/v1/eventTypes"
	v1 "github.com/Layr-Labs/protocol-apis/gen/protos/eigenlayer/sidecar/v1/events"
	"github.com/Layr-Labs/sidecar/pkg/eventBus/eventBusTypes"
	"github.com/Layr-Labs/sidecar/pkg/storage"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func (rpc *RpcServer) subscribeToBlocks(ctx context.Context, requestId string, handleBlock func(interface{}) error) error {
	consumer := &eventBusTypes.Consumer{
		Id:      eventBusTypes.ConsumerId(requestId),
		Context: context.Background(),
		Channel: make(chan *eventBusTypes.Event),
	}
	rpc.eventBus.Subscribe(consumer)
	defer rpc.eventBus.Unsubscribe(consumer)

	for {
		select {
		case <-ctx.Done():
			return nil
		case event := <-consumer.Channel:
			if event.Name == eventBusTypes.Event_BlockProcessed {
				if err := handleBlock(event.Data); err != nil {
					return err
				}
			}
		}
	}
}

func (rpc *RpcServer) StreamEigenStateChanges(request *v1.StreamEigenStateChangesRequest, g grpc.ServerStreamingServer[v1.StreamEigenStateChangesResponse]) error {
	return fmt.Errorf("not implemented")
}

func convertTransactionLogToEventTypeTransaction(log *storage.TransactionLog) *v1EventTypes.TransactionLog {
	return &v1EventTypes.TransactionLog{
		TransactionHash:  log.TransactionHash,
		TransactionIndex: log.TransactionIndex,
		LogIndex:         log.LogIndex,
		BlockNumber:      log.BlockNumber,
		Address:          log.Address,
		EventName:        log.EventName,
	}
}

func convertTransactionToEventTypeTransaction(tx *storage.Transaction) *v1EventTypes.Transaction {
	return &v1EventTypes.Transaction{
		TransactionHash:  tx.TransactionHash,
		TransactionIndex: tx.TransactionIndex,
		BlockNumber:      tx.BlockNumber,
		FromAddress:      tx.FromAddress,
		ToAddress:        tx.ToAddress,
		ContractAddress:  tx.ContractAddress,
		Logs:             nil,
	}
}

func convertBlockToEventTypeBlock(block *storage.Block) *v1EventTypes.Block {
	return &v1EventTypes.Block{
		BlockNumber: block.Number,
		BlockHash:   block.Hash,
		BlockTime:   timestamppb.New(block.BlockTime),
		ParentHash:  block.ParentHash,
	}
}

func convertBlockDataToEventTypes(blockData *eventBusTypes.BlockProcessedData) *v1EventTypes.Block {
	block := convertBlockToEventTypeBlock(blockData.Block)

	transactions := make([]*v1EventTypes.Transaction, 0)
	for _, tx := range blockData.Transactions {
		transaction := convertTransactionToEventTypeTransaction(tx)

		transactionLogs := make([]*v1EventTypes.TransactionLog, 0)
		for _, log := range blockData.Logs {
			if log.TransactionHash == tx.TransactionHash {
				transaction.Logs = append(transaction.Logs, convertTransactionLogToEventTypeTransaction(log))
			}
		}
		transaction.Logs = transactionLogs
		transactions = append(transactions, transaction)
	}
	block.Transactions = transactions
	return block
}

func (rpc *RpcServer) StreamIndexedBlocks(request *v1.StreamIndexedBlocksRequest, g grpc.ServerStreamingServer[v1.StreamIndexedBlocksResponse]) error {
	requestId, err := uuid.NewRandom()
	if err != nil {
		rpc.Logger.Error("Failed to generate request ID", zap.Error(err))
		return err
	}

	err = rpc.subscribeToBlocks(g.Context(), requestId.String(), func(data interface{}) error {
		rpc.Logger.Debug("Received block", zap.Any("data", data))
		blockProcessedData := data.(*eventBusTypes.BlockProcessedData)
		err = g.SendMsg(&v1.StreamIndexedBlocksResponse{
			Block: convertBlockDataToEventTypes(blockProcessedData),
		})
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}
