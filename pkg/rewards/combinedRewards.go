package rewards

import (
	"time"
)

const rewardsCombinedQuery = `
	with combined_rewards as (
		select
			rs.avs,
			rs.reward_hash,
			rs.token,
			rs.amount,
			rs.strategy,
			rs.strategy_index,
			rs.multiplier,
			rs.start_timestamp,
			rs.end_timestamp,
			rs.duration,
			rs.block_number,
			b.block_time,
			DATE(b.block_time) as block_date,
			rs.reward_type,
			ROW_NUMBER() OVER (PARTITION BY reward_hash, strategy_index ORDER BY block_number asc) as rn
		from reward_submissions as rs
		left join blocks as b on (b.number = rs.block_number) 
	)
	select * from combined_rewards
	where rn = 1
`

type CombinedRewards struct {
	Avs            string
	RewardHash     string
	Token          string
	Amount         string
	Strategy       string
	StrategyIndex  uint64
	Multiplier     string
	StartTimestamp time.Time
	EndTimestamp   time.Time
	Duration       uint64
	BlockNumber    uint64
	BlockDate      string
	BlockTime      time.Time
	RewardType     string // avs, all_stakers, all_earners
}

func (r *RewardsCalculator) GenerateCombinedRewards() ([]*CombinedRewards, error) {
	combinedRewards := make([]*CombinedRewards, 0)

	res := r.grm.Raw(rewardsCombinedQuery).Scan(&combinedRewards)
	if res.Error != nil {
		r.logger.Sugar().Errorw("Failed to generate combined rewards", "error", res.Error)
		return nil, res.Error
	}
	return combinedRewards, nil
}

func (r *RewardsCalculator) GenerateAndInsertCombinedRewards() error {
	combinedRewards, err := r.GenerateCombinedRewards()
	if err != nil {
		r.logger.Sugar().Errorw("Failed to generate combined rewards", "error", err)
		return err
	}

	r.logger.Sugar().Infow("Inserting combined rewards", "count", len(combinedRewards))
	res := r.grm.Model(&CombinedRewards{}).CreateInBatches(combinedRewards, 500)
	if res.Error != nil {
		r.logger.Sugar().Errorw("Failed to insert combined rewards", "error", res.Error)
		return res.Error
	}
	return nil
}
