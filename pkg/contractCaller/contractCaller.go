package contractCaller

import (
	"context"
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
}
