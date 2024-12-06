package rewardsCalculatorQueue

import (
	"context"
	"github.com/Layr-Labs/sidecar/pkg/rewards"
	"go.uber.org/zap"
)

// NewRewardsCalculatorQueue creates a new RewardsCalculatorQueue
func NewRewardsCalculatorQueue(rc *rewards.RewardsCalculator, logger *zap.Logger) *RewardsCalculatorQueue {
	return &RewardsCalculatorQueue{
		logger:            logger,
		rewardsCalculator: rc,
		queue:             make(chan *RewardsCalculationMessage),
	}
}

// Enqueue adds a new message to the queue and returns immediately
func (rcq *RewardsCalculatorQueue) Enqueue(payload *RewardsCalculationMessage) {
	rcq.logger.Sugar().Infow("Enqueueing rewards calculation message", "data", payload.Data)
	rcq.queue <- payload
}

// EnqueueAndWait adds a new message to the queue and waits for a response or returns if the context is done
func (rcq *RewardsCalculatorQueue) EnqueueAndWait(ctx context.Context, data RewardsCalculationData) (*RewardsCalculatorResponseData, error) {
	responseChan := make(chan *RewardsCalculatorResponse)

	payload := &RewardsCalculationMessage{
		Data:         data,
		ResponseChan: responseChan,
	}
	rcq.Enqueue(payload)

	rcq.logger.Sugar().Infow("Waiting for rewards calculation response", "data", data)

	select {
	case response := <-responseChan:
		return response.Data, response.Error
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (rcq *RewardsCalculatorQueue) Close() {
	close(rcq.done)
}
