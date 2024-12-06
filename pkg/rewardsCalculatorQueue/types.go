package rewardsCalculatorQueue

import (
	"github.com/Layr-Labs/sidecar/pkg/rewards"
	"go.uber.org/zap"
)

type RewardsCalculationType string

var (
	RewardsCalculationType_CalculateRewards                RewardsCalculationType = "calculateRewards"
	RewardsCalculationType_BackfillStakerOperators         RewardsCalculationType = "backfillStakerOperators"
	RewardsCalculationType_BackfillStakerOperatorsSnapshot RewardsCalculationType = "backfillStakerOperatorsSnapshot"
)

type RewardsCalculationData struct {
	CalculationType RewardsCalculationType
	CutoffDate      string
}

type RewardsCalculationMessage struct {
	Data         RewardsCalculationData
	ResponseChan chan *RewardsCalculatorResponse
}

type RewardsCalculatorResponseData struct {
	CutoffDate string
}

type RewardsCalculatorResponse struct {
	Data  *RewardsCalculatorResponseData
	Error error
}

type RewardsCalculatorQueue struct {
	logger            *zap.Logger
	rewardsCalculator *rewards.RewardsCalculator
	queue             chan *RewardsCalculationMessage
	done              chan struct{}
}
