package rpcServer

import (
	"context"

	sidecarV1 "github.com/Layr-Labs/protocol-apis/gen/protos/eigenlayer/sidecar/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (rpc *RpcServer) GetBlockHeight(ctx context.Context, req *sidecarV1.GetBlockHeightRequest) (*sidecarV1.GetBlockHeightResponse, error) {
	verified := req.GetVerified()
	if verified {
		sr, err := rpc.stateManager.GetLatestStateRoot()
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
		block, err := rpc.blockStore.GetBlockByNumber(sr.EthBlockNumber)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
		return &sidecarV1.GetBlockHeightResponse{
			BlockNumber: block.Number,
			BlockHash:   block.Hash,
		}, nil
	}

	block, err := rpc.blockStore.GetLatestBlock()

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &sidecarV1.GetBlockHeightResponse{
		BlockNumber: block.Number,
		BlockHash:   block.Hash,
	}, nil
}

func (rpc *RpcServer) GetStateRoot(ctx context.Context, req *sidecarV1.GetStateRootRequest) (*sidecarV1.GetStateRootResponse, error) {
	blockNumber := req.GetBlockNumber()
	stateRoot, err := rpc.stateManager.GetStateRootForBlock(blockNumber)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &sidecarV1.GetStateRootResponse{
		EthBlockHash:   stateRoot.EthBlockHash,
		EthBlockNumber: stateRoot.EthBlockNumber,
		StateRoot:      stateRoot.StateRoot,
	}, nil
}
