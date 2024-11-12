package contractCaller

import (
	"context"
	"github.com/Layr-Labs/eigenlayer-contracts/pkg/bindings/IRewardsCoordinator"
	"github.com/ethereum/go-ethereum/common"
)

type OperatorRestakedStrategy struct {
	Operator string
	Avs      string
	Results  []common.Address
}

type IContractCaller interface {
	GetOperatorRestakedStrategies(ctx context.Context, avs string, operator string, blockNumber uint64) ([]common.Address, error)
	GetAllOperatorRestakedStrategies(ctx context.Context, operatorRestakedStrategies []*OperatorRestakedStrategy, blockNumber uint64) ([]*OperatorRestakedStrategy, error)
	GetDistributionRootByIndex(ctx context.Context, index uint64) (*IRewardsCoordinator.IRewardsCoordinatorDistributionRoot, error)
}
