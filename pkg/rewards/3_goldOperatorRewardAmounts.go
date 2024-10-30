package rewards

import (
	"go.uber.org/zap"
)

const _3_goldOperatorRewardAmountsQuery = `
create table {{.destTableName}} as
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
  FROM {{.stakerRewardAmountsTable}}
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
`

func (rc *RewardsCalculator) GenerateGold3OperatorRewardAmountsTable(startDate string, snapshotDate string) error {
	allTableNames := getGoldTableNames(snapshotDate)
	destTableName := allTableNames[Table_3_OperatorRewardAmounts]

	rc.logger.Sugar().Infow("Generating staker reward amounts",
		zap.String("startDate", startDate),
		zap.String("cutoffDate", snapshotDate),
		zap.String("destTableName", destTableName),
	)

	query, err := renderQueryTemplate(_3_goldOperatorRewardAmountsQuery, map[string]string{
		"destTableName":            destTableName,
		"stakerRewardAmountsTable": allTableNames[Table_2_StakerRewardAmounts],
	})
	if err != nil {
		rc.logger.Sugar().Errorw("Failed to render query template", "error", err)
		return err
	}

	res := rc.grm.Exec(query)
	if res.Error != nil {
		rc.logger.Sugar().Errorw("Failed to create gold_operator_reward_amounts", "error", res.Error)
		return res.Error
	}
	return nil
}
