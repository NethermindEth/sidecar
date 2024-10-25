package rewards

import (
	"database/sql"
	"github.com/Layr-Labs/go-sidecar/internal/config"
)

const _2_goldStakerRewardAmountsQuery = `
insert into gold_2_staker_reward_amounts
WITH reward_snapshot_operators as (
  SELECT
    ap.reward_hash,
    ap.snapshot::date as snapshot,
    ap.token,
    ap.tokens_per_day,
    ap.tokens_per_day_decimal,
    ap.avs,
    ap.strategy,
    ap.multiplier,
    ap.reward_type,
    ap.reward_submission_date,
    oar.operator
  FROM gold_1_active_rewards ap
  JOIN operator_avs_registration_snapshots oar
  ON ap.avs = oar.avs and ap.snapshot = oar.snapshot
  WHERE ap.reward_type = 'avs'
),
_operator_restaked_strategies AS (
  SELECT
    rso.*
  FROM reward_snapshot_operators rso
  JOIN operator_avs_strategy_snapshots oas
  ON
    rso.operator = oas.operator AND
    rso.avs = oas.avs AND
    rso.strategy = oas.strategy AND
    rso.snapshot = oas.snapshot
),
-- Get the stakers that were delegated to the operator for the snapshot
staker_delegated_operators AS (
  SELECT
    ors.*,
    sds.staker
  FROM _operator_restaked_strategies ors
  JOIN staker_delegation_snapshots sds
  ON
    ors.operator = sds.operator AND
    ors.snapshot = sds.snapshot
),
-- Get the shares for staker delegated to the operator
staker_avs_strategy_shares AS (
  SELECT
    sdo.*,
    sss.shares
  FROM staker_delegated_operators sdo
  JOIN staker_share_snapshots sss
  ON
    sdo.staker = sss.staker AND
    sdo.snapshot = sss.snapshot AND
    sdo.strategy = sss.strategy
  -- Parse out negative shares and zero multiplier so there is no division by zero case
  WHERE sss.shares > 0 and sdo.multiplier != 0
),
-- Calculate the weight of a staker
staker_weights AS (
  SELECT *,
    SUM(multiplier * shares) OVER (PARTITION BY staker, reward_hash, snapshot) AS staker_weight
  FROM staker_avs_strategy_shares
),
-- Get distinct stakers since their weights are already calculated
distinct_stakers AS (
  SELECT *
  FROM (
      SELECT *,
        -- We can use an arbitrary order here since the staker_weight is the same for each (staker, strategy, hash, snapshot)
        -- We use strategy ASC for better debuggability
        ROW_NUMBER() OVER (PARTITION BY reward_hash, snapshot, staker ORDER BY strategy ASC) as rn
      FROM staker_weights
  ) t
  WHERE rn = 1
  ORDER BY reward_hash, snapshot, staker
),
-- Calculate sum of all staker weights for each reward and snapshot
staker_weight_sum AS (
  SELECT *,
    SUM(staker_weight) OVER (PARTITION BY reward_hash, snapshot) as total_weight
  FROM distinct_stakers
),
-- Calculate staker proportion of tokens for each reward and snapshot
staker_proportion AS (
  SELECT *,
    FLOOR((staker_weight / total_weight) * 1000000000000000) / 1000000000000000 AS staker_proportion
  FROM staker_weight_sum
),
-- Calculate total tokens to the (staker, operator) pair
staker_operator_total_tokens AS (
  SELECT *,
    CASE
      -- For snapshots that are before the hard fork AND submitted before the hard fork, we use the old calc method
      WHEN snapshot < @amazonHardforkDate AND reward_submission_date < @amazonHardforkDate THEN
        cast(staker_proportion * tokens_per_day AS DECIMAL(38,0))
      WHEN snapshot < @nileHardforkDate AND reward_submission_date < @nileHardforkDate THEN
        (staker_proportion * tokens_per_day)::text::decimal(38,0)
      ELSE
        FLOOR(staker_proportion * tokens_per_day_decimal)
    END as total_staker_operator_payout
  FROM staker_proportion
),
-- Calculate the token breakdown for each (staker, operator) pair
token_breakdowns AS (
  SELECT *,
    CASE
      WHEN snapshot < @amazonHardforkDate AND reward_submission_date < @amazonHardforkDate THEN
        cast(total_staker_operator_payout * 0.10 AS DECIMAL(38,0))
      WHEN snapshot < @nileHardforkDate AND reward_submission_date < @nileHardforkDate THEN
        (total_staker_operator_payout * 0.10)::text::decimal(38,0)
      ELSE
        floor(total_staker_operator_payout * 0.10)
    END as operator_tokens,
    CASE
      WHEN snapshot < @amazonHardforkDate AND reward_submission_date < @amazonHardforkDate THEN
        total_staker_operator_payout - cast(total_staker_operator_payout * 0.10 as DECIMAL(38,0))
      WHEN snapshot < @nileHardforkDate AND reward_submission_date < @nileHardforkDate THEN
        total_staker_operator_payout - ((total_staker_operator_payout * 0.10)::text::decimal(38,0))
      ELSE
        total_staker_operator_payout - floor(total_staker_operator_payout * 0.10)
    END as staker_tokens
  FROM staker_operator_total_tokens
)
SELECT * from token_breakdowns
where
	DATE(snapshot) >= @startDate
	and DATE(snapshot) < @cutoffDate
ORDER BY reward_hash, snapshot, staker, operator
`

func (rc *RewardsCalculator) GenerateGold2StakerRewardAmountsTable(startDate string, snapshotDate string, forks config.ForkMap) error {
	res := rc.grm.Exec(_2_goldStakerRewardAmountsQuery,
		sql.Named("startDate", startDate),
		sql.Named("cutoffDate", snapshotDate),
		sql.Named("amazonHardforkDate", forks[config.Fork_Amazon]),
		sql.Named("nileHardforkDate", forks[config.Fork_Nile]),
	)
	if res.Error != nil {
		rc.logger.Sugar().Errorw("Failed to create gold_staker_reward_amounts", "error", res.Error)
		return res.Error
	}
	return nil
}
