package storage

import (
	"math/big"
	"time"
)

// ----------------------------------------------------------------------------
// Append only tables of state
// ----------------------------------------------------------------------------

/*
create table if not exists staker_share_changes (

	id serial primary key,
	staker varchar,
	strategy varchar,
	shares numeric,
	transaction_hash varchar,
	log_index bigint,
	block_number bigint,
	created_at timestamp with time zone

);.
*/
type StakerShareChanges struct {
	Id              uint64 `gorm:"type:serial"`
	Staker          string
	Strategy        string
	Shares          big.Int `gorm:"type:numeric"`
	TransactionHash string
	LogIndex        uint64
	BlockNumber     uint64
	CreatedAt       time.Time
}

/*
create table if not exists active_reward_submissions (

	id serial primary key,
	avs varchar,
	reward_hash varchar,
	token varchar,
	amount numeric,
	strategy varchar,
	multiplier numeric,
	strategy_index bigint,
	transaction_hash varchar,
	log_index bigint,
	block_number bigint,
	start_timestamp timestamp,
	end_timestamp timestamp,
	duration bigint
	created_at timestamp with time zone

);.
*/
type ActiveRewardSubmissions struct {
	Id              uint64 `gorm:"type:serial"`
	Avs             string
	RewardHash      string
	Token           string
	Amount          big.Int `gorm:"type:numeric"`
	Strategy        string
	Multiplier      big.Int `gorm:"type:numeric"`
	StrategyIndex   uint64
	TransactionHash string
	LogIndex        uint64
	BlockNumber     uint64
	StartTimestamp  time.Time
	EndTimestamp    time.Time
	Duration        uint64
	CreatedAt       time.Time
}

/*
create table if not exists active_reward_for_all_submissions (

	id serial primary key,
	avs varchar,
	reward_hash varchar,
	token varchar,
	amount numeric,
	strategy varchar,
	multiplier numeric,
	strategy_index bigint,
	transaction_hash varchar,
	log_index bigint,
	block_number bigint,
	start_timestamp timestamp,
	end_timestamp timestamp,
	duration bigint
	created_at timestamp with time zone

);.
*/
type RewardForAllSubmissions struct {
	Id              uint64 `gorm:"type:serial"`
	Avs             string
	RewardHash      string
	Token           string
	Amount          big.Int `gorm:"type:numeric"`
	Strategy        string
	Multiplier      big.Int `gorm:"type:numeric"`
	StrategyIndex   uint64
	TransactionHash string
	LogIndex        uint64
	BlockNumber     uint64
	StartTimestamp  time.Time
	EndTimestamp    time.Time
	Duration        uint64
	CreatedAt       time.Time
}

// ----------------------------------------------------------------------------
// Block-based "summary" tables
// ----------------------------------------------------------------------------
/*
create table if not exists active_rewards (

	avs varchar,
	reward_hash varchar,
	token varchar,
	amount numeric,
	strategy varchar,
	multiplier numeric,
	strategy_index bigint,
	block_number bigint,
	start_timestamp timestamp,
	end_timestamp timestamp,
	duration bigint,
	created_at timestamp with time zone

).
*/
type ActiveRewards struct {
	Avs            string
	RewardHash     string
	Token          string
	Amount         big.Int `gorm:"type:numeric"`
	Strategy       string
	Multiplier     big.Int `gorm:"type:numeric"`
	StrategyIndex  uint64
	BlockNumber    uint64
	StartTimestamp time.Time
	EndTimestamp   time.Time
	Duration       uint64
	CreatedAt      time.Time
}

/*
create table if not exists active_reward_for_all (

	avs varchar,
	reward_hash varchar,
	token varchar,
	amount numeric,
	strategy varchar,
	multiplier numeric,
	strategy_index bigint,
	block_number bigint,
	start_timestamp timestamp,
	end_timestamp timestamp,
	duration bigint,
	created_at timestamp with time zone

).
*/
type ActiveRewardForAll struct {
	Avs            string
	RewardHash     string
	Token          string
	Amount         big.Int `gorm:"type:numeric"`
	Strategy       string
	Multiplier     big.Int `gorm:"type:numeric"`
	StrategyIndex  uint64
	BlockNumber    uint64
	StartTimestamp time.Time
	EndTimestamp   time.Time
	Duration       uint64
	CreatedAt      time.Time
}
