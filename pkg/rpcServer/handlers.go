package rpcServer

import (
	"context"

	sidecarV1 "github.com/Layr-Labs/protocol-apis/gen/protos/eigenlayer/sidecar/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (rpc *RpcServer) GetBlockHeight(context.Context, *sidecarV1.GetBlockHeightRequest) (*sidecarV1.GetBlockHeightResponse, error) {
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

func (rpc *RpcServer) GetRewardsRoot(ctx context.Context, req *sidecarV1.GetRewardsRootRequest) (*sidecarV1.GetRewardsRootResponse, error) {
	return nil, status.Error(codes.Unimplemented, "method GetRewardsRoot not implemented")
}

func (rpc *RpcServer) GenerateClaimProof(ctx context.Context, req *sidecarV1.GenerateClaimProofRequest) (*sidecarV1.GenerateClaimProofResponse, error) {
	return nil, status.Error(codes.Unimplemented, "method GenerateClaimProof not implemented")
}

func (rpc *RpcServer) GetAvailableRewards(ctx context.Context, req *sidecarV1.GetAvailableRewardsRequest) (*sidecarV1.GetAvailableRewardsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "method GetAvailableRewards not implemented")
}

func (rpc *RpcServer) GetAvailableRewardsTokens(ctx context.Context, req *sidecarV1.GetAvailableRewardsTokensRequest) (*sidecarV1.GetAvailableRewardsTokensResponse, error) {
	return nil, status.Error(codes.Unimplemented, "method GetAvailableRewardsTokens not implemented")
}
