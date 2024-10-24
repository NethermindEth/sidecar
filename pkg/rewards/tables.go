package rewards

import "time"

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

type OperatorAvsRegistrationSnapshots struct {
	Avs      string
	Operator string
	Snapshot string
}

type OperatorAvsStrategySnapshot struct {
	Operator string
	Avs      string
	Strategy string
	Snapshot time.Time
}
