package rpcServer

import (
	"context"
	"fmt"
	v1EventTypes "github.com/Layr-Labs/protocol-apis/gen/protos/eigenlayer/sidecar/v1/eventTypes"
	v1 "github.com/Layr-Labs/protocol-apis/gen/protos/eigenlayer/sidecar/v1/events"
	"github.com/Layr-Labs/sidecar/pkg/eigenState/avsOperators"
	"github.com/Layr-Labs/sidecar/pkg/eigenState/disabledDistributionRoots"
	"github.com/Layr-Labs/sidecar/pkg/eigenState/operatorShares"
	"github.com/Layr-Labs/sidecar/pkg/eigenState/rewardSubmissions"
	"github.com/Layr-Labs/sidecar/pkg/eigenState/stakerDelegations"
	"github.com/Layr-Labs/sidecar/pkg/eigenState/stakerShares"
	"github.com/Layr-Labs/sidecar/pkg/eigenState/stateManager"
	"github.com/Layr-Labs/sidecar/pkg/eigenState/submittedDistributionRoots"
	"github.com/Layr-Labs/sidecar/pkg/eigenState/types"
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
		Context: ctx,
		Channel: make(chan *eventBusTypes.Event),
	}
	rpc.eventBus.Subscribe(consumer)
	defer rpc.eventBus.Unsubscribe(consumer)

	for {
		select {
		case <-ctx.Done():
			rpc.Logger.Sugar().Info("Context done, exiting subscription", zap.String("requestId", requestId))
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
	requestId, err := uuid.NewRandom()
	if err != nil {
		rpc.Logger.Error("Failed to generate request ID", zap.Error(err))
		return err
	}

	err = rpc.subscribeToBlocks(g.Context(), requestId.String(), func(data interface{}) error {
		blockProcessedData := data.(*eventBusTypes.BlockProcessedData)
		changes, err := rpc.parseCommittedChanges(blockProcessedData.CommittedState)
		if err != nil {
			return err
		}
		return g.SendMsg(&v1.StreamEigenStateChangesResponse{
			BlockNumber: blockProcessedData.Block.Number,
			StateRoot:   convertStateRootToEventTypeStateRoot(blockProcessedData.StateRoot),
			Changes:     changes,
		})
	})
	return err
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

		resp, err := rpc.buildBlockResponse(blockProcessedData)
		if err != nil {
			return err
		}

		return g.SendMsg(resp)
	})
	if err != nil {
		return err
	}
	return nil
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

func convertStateRootToEventTypeStateRoot(stateRoot *stateManager.StateRoot) *v1EventTypes.StateRoot {
	return &v1EventTypes.StateRoot{
		EthBlockNumber: stateRoot.EthBlockNumber,
		EthBlockHash:   stateRoot.EthBlockHash,
		StateRoot:      stateRoot.StateRoot,
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

func (rpc *RpcServer) buildBlockResponse(blockData *eventBusTypes.BlockProcessedData) (*v1.StreamIndexedBlocksResponse, error) {
	changes, err := rpc.parseCommittedChanges(blockData.CommittedState)
	if err != nil {
		return nil, err
	}
	return &v1.StreamIndexedBlocksResponse{
		Block:     convertBlockDataToEventTypes(blockData),
		StateRoot: convertStateRootToEventTypeStateRoot(blockData.StateRoot),
		Changes:   changes,
	}, nil
}

func convertAvsOperatorToStateChange(change interface{}) *v1EventTypes.AvsOperatorStateChange {
	typedChange := change.(*avsOperators.AvsOperatorStateChange)
	return &v1EventTypes.AvsOperatorStateChange{
		Avs:        typedChange.Avs,
		Operator:   typedChange.Operator,
		Registered: typedChange.Registered,
		TransactionMetadata: &v1EventTypes.TransactionMetadata{
			TransactionHash: typedChange.TransactionHash,
			LogIndex:        typedChange.LogIndex,
			BlockHeight:     typedChange.BlockNumber,
		},
	}
}

func convertDisabledDistributionRootToStateChange(change interface{}) *v1EventTypes.DisabledDistributionRoot {
	typedChange := change.(*types.DisabledDistributionRoot)
	return &v1EventTypes.DisabledDistributionRoot{
		RootIndex: typedChange.RootIndex,
		TransactionMetadata: &v1EventTypes.TransactionMetadata{
			TransactionHash: typedChange.TransactionHash,
			LogIndex:        typedChange.LogIndex,
			BlockHeight:     typedChange.BlockNumber,
		},
	}
}

func convertOperatorSharesToStateChange(change interface{}) *v1EventTypes.OperatorShareDelta {
	typedChange := change.(*operatorShares.OperatorShareDeltas)
	return &v1EventTypes.OperatorShareDelta{
		Operator: typedChange.Operator,
		Shares:   typedChange.Shares,
		TransactionMetadata: &v1EventTypes.TransactionMetadata{
			TransactionHash: typedChange.TransactionHash,
			LogIndex:        typedChange.LogIndex,
			BlockHeight:     typedChange.BlockNumber,
		},
	}
}

func convertRewardTypeToStateChange(rewardType string) (v1EventTypes.RewardSubmission_RewardType, error) {
	switch rewardType {
	case "avs":
		return v1EventTypes.RewardSubmission_AVS, nil
	case "all_stakers":
		return v1EventTypes.RewardSubmission_ALL_STAKERS, nil
	case "all_earners":
		return v1EventTypes.RewardSubmission_ALL_EARNERS, nil
	}
	return -1, fmt.Errorf("Invalid reward type '%s'", rewardType)
}

func convertRewardSubmissionToStateChange(change interface{}) (*v1EventTypes.RewardSubmission, error) {
	typedChange := change.(*rewardSubmissions.RewardSubmission)
	rewardType, err := convertRewardTypeToStateChange(typedChange.RewardType)
	if err != nil {
		return nil, err
	}
	return &v1EventTypes.RewardSubmission{
		Avs:            typedChange.Avs,
		RewardHash:     typedChange.RewardHash,
		Token:          typedChange.Token,
		Amount:         typedChange.Amount,
		Strategy:       typedChange.Strategy,
		StrategyIndex:  typedChange.StrategyIndex,
		Multiplier:     typedChange.Multiplier,
		StartTimestamp: timestamppb.New(*typedChange.StartTimestamp),
		EndTimestamp:   timestamppb.New(*typedChange.EndTimestamp),
		Duration:       typedChange.Duration,
		RewardType:     rewardType,
		TransactionMetadata: &v1EventTypes.TransactionMetadata{
			TransactionHash: typedChange.TransactionHash,
			LogIndex:        typedChange.LogIndex,
			BlockHeight:     typedChange.BlockNumber,
		},
	}, nil
}

func convertStakerDelegationToStateChange(change interface{}) *v1EventTypes.StakerDelegationChange {
	typedChange := change.(*stakerDelegations.StakerDelegationChange)
	return &v1EventTypes.StakerDelegationChange{
		Staker:    typedChange.Staker,
		Operator:  typedChange.Operator,
		Delegated: typedChange.Delegated,
		TransactionMetadata: &v1EventTypes.TransactionMetadata{
			TransactionHash: typedChange.TransactionHash,
			LogIndex:        typedChange.LogIndex,
			BlockHeight:     typedChange.BlockNumber,
		},
	}
}

func convertStakerSharesToStateChange(change interface{}) *v1EventTypes.StakerShareDelta {
	typedChange := change.(*stakerShares.StakerShareDeltas)
	return &v1EventTypes.StakerShareDelta{
		Staker:        typedChange.Staker,
		Strategy:      typedChange.Strategy,
		Shares:        typedChange.Shares,
		StrategyIndex: typedChange.StrategyIndex,
		BlockTime:     timestamppb.New(typedChange.BlockTime),
		BlockDate:     typedChange.BlockDate,
		TransactionMetadata: &v1EventTypes.TransactionMetadata{
			TransactionHash: typedChange.TransactionHash,
			LogIndex:        typedChange.LogIndex,
			BlockHeight:     typedChange.BlockNumber,
		},
	}
}

func convertSubmittedDistributionRootToStateChange(change interface{}) *v1EventTypes.SubmittedDistributionRoot {
	typedChange := change.(*types.SubmittedDistributionRoot)
	return &v1EventTypes.SubmittedDistributionRoot{
		Root:                      typedChange.Root,
		RootIndex:                 typedChange.RootIndex,
		RewardsCalculationEnd:     timestamppb.New(typedChange.RewardsCalculationEnd),
		RewardsCalculationEndUnit: typedChange.RewardsCalculationEndUnit,
		ActivatedAt:               timestamppb.New(typedChange.ActivatedAt),
		ActivatedAtUnit:           typedChange.ActivatedAtUnit,
		CreatedAtBlockNumber:      typedChange.CreatedAtBlockNumber,
		TransactionMetadata: &v1EventTypes.TransactionMetadata{
			BlockHeight:     typedChange.BlockNumber,
			TransactionHash: typedChange.TransactionHash,
			LogIndex:        typedChange.LogIndex,
		},
	}
}

func (rpc *RpcServer) parseCommittedChanges(committedStateByModel map[string][]interface{}) ([]*v1EventTypes.EigenStateChange, error) {
	parsedChanges := make([]*v1EventTypes.EigenStateChange, 0)

	for modelName, changes := range committedStateByModel {
		for _, change := range changes {
			switch modelName {
			case avsOperators.AvsOperatorsModelName:
				parsedChanges = append(parsedChanges, &v1EventTypes.EigenStateChange{
					Change: &v1EventTypes.EigenStateChange_AvsOperatorStateChange{
						AvsOperatorStateChange: convertAvsOperatorToStateChange(change),
					},
				})
			case disabledDistributionRoots.DisabledDistributionRootsModelName:
				parsedChanges = append(parsedChanges, &v1EventTypes.EigenStateChange{
					Change: &v1EventTypes.EigenStateChange_DisabledDistributionRoot{
						DisabledDistributionRoot: convertDisabledDistributionRootToStateChange(change),
					},
				})
			case operatorShares.OperatorSharesModelName:
				parsedChanges = append(parsedChanges, &v1EventTypes.EigenStateChange{
					Change: &v1EventTypes.EigenStateChange_OperatorShareDelta{
						OperatorShareDelta: convertOperatorSharesToStateChange(change),
					},
				})
			case rewardSubmissions.RewardSubmissionsModelName:
				parsedChange, err := convertRewardSubmissionToStateChange(change)
				if err != nil {
					return nil, err
				}

				parsedChanges = append(parsedChanges, &v1EventTypes.EigenStateChange{
					Change: &v1EventTypes.EigenStateChange_RewardSubmission{
						RewardSubmission: parsedChange,
					},
				})
			case stakerDelegations.StakerDelegationsModelName:
				parsedChanges = append(parsedChanges, &v1EventTypes.EigenStateChange{
					Change: &v1EventTypes.EigenStateChange_StakerDelegationChange{
						StakerDelegationChange: convertStakerDelegationToStateChange(change),
					},
				})
			case stakerShares.StakerSharesModelName:
				parsedChanges = append(parsedChanges, &v1EventTypes.EigenStateChange{
					Change: &v1EventTypes.EigenStateChange_StakerShareDelta{
						StakerShareDelta: convertStakerSharesToStateChange(change),
					},
				})
			case submittedDistributionRoots.SubmittedDistributionRootsModelName:
				parsedChanges = append(parsedChanges, &v1EventTypes.EigenStateChange{
					Change: &v1EventTypes.EigenStateChange_SubmittedDistributionRoot{
						SubmittedDistributionRoot: convertSubmittedDistributionRootToStateChange(change),
					},
				})
			}
		}
	}
	return parsedChanges, nil
}
