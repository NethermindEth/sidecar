package rpcServer

import (
	"context"
	"github.com/Layr-Labs/eigenlayer-rewards-proofs/pkg/claimgen"
	rewardsV1 "github.com/Layr-Labs/protocol-apis/gen/protos/eigenlayer/sidecar/v1/rewards"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func convertClaimProofToRPCResponse(solidityProof *claimgen.IRewardsCoordinatorRewardsMerkleClaimStrings) *rewardsV1.Proof {
	tokenLeaves := make([]*rewardsV1.TokenLeaf, 0)

	for _, l := range solidityProof.TokenLeaves {
		tokenLeaves = append(tokenLeaves, &rewardsV1.TokenLeaf{
			Token:              l.Token.String(),
			CumulativeEarnings: l.CumulativeEarnings,
		})
	}

	return &rewardsV1.Proof{
		Root:            solidityProof.Root,
		RootIndex:       solidityProof.RootIndex,
		EarnerIndex:     solidityProof.EarnerIndex,
		EarnerTreeProof: solidityProof.EarnerTreeProof,
		EarnerLeaf: &rewardsV1.EarnerLeaf{
			Earner:          solidityProof.EarnerLeaf.Earner.String(),
			EarnerTokenRoot: solidityProof.EarnerLeaf.EarnerTokenRoot,
		},
		LeafIndices:     solidityProof.TokenIndices,
		TokenTreeProofs: solidityProof.TokenTreeProofs,
		TokenLeaves:     tokenLeaves,
	}
}

func (rpc *RpcServer) GenerateClaimProof(ctx context.Context, req *rewardsV1.GenerateClaimProofRequest) (*rewardsV1.GenerateClaimProofResponse, error) {
	earner := req.GetEarnerAddress()
	tokens := req.GetTokens()
	rootIndex := req.GetRootIndex()

	var rootIndexVal int64
	if rootIndex == nil {
		rootIndexVal = -1
	} else {
		rootIndexVal = rootIndex.GetValue()
	}

	root, claim, err := rpc.rewardsProofs.GenerateRewardsClaimProof(earner, tokens, rootIndexVal)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to generate claim proof %s", err.Error())
	}

	solidityClaim := claimgen.FormatProofForSolidity(root, claim)

	return &rewardsV1.GenerateClaimProofResponse{
		Proof: convertClaimProofToRPCResponse(solidityClaim),
	}, nil
}
