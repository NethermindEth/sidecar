package rewards

import "database/sql"

const _3_goldOperatorRewardAmountsQuery = `
insert into gold_3_operator_reward_amounts
WITH operator_token_sums AS (
  SELECT
    reward_hash,
    snapshot,
    token,
    tokens_per_day,
    avs,
    strategy,
    multiplier,
    reward_type,
    operator,
    SUM(operator_tokens) OVER (PARTITION BY operator, reward_hash, snapshot) AS operator_tokens
  FROM gold_2_staker_reward_amounts
),
-- Dedupe the operator tokens across strategies for each operator, reward hash, and snapshot
distinct_operators AS (
  SELECT *
  FROM (
      SELECT *,
        -- We can use an arbitrary order here since the staker_weight is the same for each (operator, strategy, hash, snapshot)
        -- We use strategy ASC for better debuggability
        ROW_NUMBER() OVER (PARTITION BY reward_hash, snapshot, operator ORDER BY strategy ASC) as rn
      FROM operator_token_sums
  ) t
  WHERE rn = 1
)
SELECT * FROM distinct_operators
where
	snapshot >= @startDate
	and snapshot < @cutoffDate
`

func (rc *RewardsCalculator) GenerateGold3OperatorRewardAmountsTable(startDate string, snapshotDate string) error {
	res := rc.grm.Exec(_3_goldOperatorRewardAmountsQuery,
		sql.Named("startDate", startDate),
		sql.Named("cutoffDate", snapshotDate),
	)
	if res.Error != nil {
		rc.logger.Sugar().Errorw("Failed to create gold_operator_reward_amounts", "error", res.Error)
		return res.Error
	}
	return nil
}
