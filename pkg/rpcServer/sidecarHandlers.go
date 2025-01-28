package rpcServer

import (
	"context"
	"github.com/Layr-Labs/sidecar/internal/version"

	sidecarV1 "github.com/Layr-Labs/protocol-apis/gen/protos/eigenlayer/sidecar/v1/sidecar"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (rpc *RpcServer) GetBlockHeight(ctx context.Context, req *sidecarV1.GetBlockHeightRequest) (*sidecarV1.GetBlockHeightResponse, error) {
	verified := req.GetVerified()
	block, err := rpc.protocolDataService.GetCurrentBlockHeight(ctx, verified)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	if block == nil {
		return nil, status.Error(codes.NotFound, "Block not found")
	}
	return &sidecarV1.GetBlockHeightResponse{
		BlockNumber: block.Number,
		BlockHash:   block.Hash,
	}, nil
}

func (rpc *RpcServer) GetStateRoot(ctx context.Context, req *sidecarV1.GetStateRootRequest) (*sidecarV1.GetStateRootResponse, error) {
	blockNumber := req.GetBlockNumber()
	stateRoot, err := rpc.protocolDataService.GetStateRoot(ctx, blockNumber)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &sidecarV1.GetStateRootResponse{
		EthBlockHash:   stateRoot.EthBlockHash,
		EthBlockNumber: stateRoot.EthBlockNumber,
		StateRoot:      stateRoot.StateRoot,
	}, nil
}

func (rpc *RpcServer) About(ctx context.Context, req *sidecarV1.AboutRequest) (*sidecarV1.AboutResponse, error) {
	return &sidecarV1.AboutResponse{
		Version: version.GetVersion(),
		Commit:  version.GetCommit(),
		Chain:   rpc.globalConfig.Chain.String(),
	}, nil
}
