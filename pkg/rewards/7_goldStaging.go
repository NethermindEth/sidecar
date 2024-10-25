package rewards

import "database/sql"

const _7_goldStagingQuery = `
insert into gold_7_staging
WITH staker_rewards AS (
  -- We can select DISTINCT here because the staker's tokens are the same for each strategy in the reward hash
  SELECT DISTINCT
    staker as earner,
    snapshot,
    reward_hash,
    token,
    staker_tokens as amount
  FROM gold_2_staker_reward_amounts
),
operator_rewards AS (
  SELECT DISTINCT
    -- We can select DISTINCT here because the operator's tokens are the same for each strategy in the reward hash
    operator as earner,
    snapshot,
    reward_hash,
    token,
    operator_tokens as amount
  FROM gold_3_operator_reward_amounts
),
rewards_for_all AS (
  SELECT DISTINCT
    staker as earner,
    snapshot,
    reward_hash,
    token,
    staker_tokens as amount
  FROM gold_4_rewards_for_all
),
rewards_for_all_earners_stakers AS (
  SELECT DISTINCT
    staker as earner,
    snapshot,
    reward_hash,
    token,
    staker_tokens as amounts
  FROM gold_5_rfae_stakers
),
rewards_for_all_earners_operators AS (
  SELECT DISTINCT
    operator as earner,
    snapshot,
    reward_hash,
    token,
    operator_tokens as amount
  FROM gold_6_rfae_operators
),
combined_rewards AS (
  SELECT * FROM operator_rewards
  UNION ALL
  SELECT * FROM staker_rewards
  UNION ALL
  SELECT * FROM rewards_for_all
  UNION ALL
  SELECT * FROM rewards_for_all_earners_stakers
  UNION ALL
  SELECT * FROM rewards_for_all_earners_operators
),
-- Dedupe earners, primarily operators who are also their own staker.
deduped_earners AS (
  SELECT
    earner,
    snapshot,
    reward_hash,
    token,
    SUM(amount) as amount
  FROM combined_rewards
  GROUP BY
    earner,
    snapshot,
    reward_hash,
    token
)
SELECT *
FROM deduped_earners
where
	DATE(snapshot) >= @startDate
	and DATE(snapshot) < @cutoffDate
`

func (rc *RewardsCalculator) GenerateGold7StagingTable(startDate string, snapshotDate string) error {
	res := rc.grm.Exec(_7_goldStagingQuery,
		sql.Named("startDate", startDate),
		sql.Named("cutoffDate", snapshotDate),
	)
	if res.Error != nil {
		rc.logger.Sugar().Errorw("Failed to create gold_staging", "error", res.Error)
		return res.Error
	}
	return nil
}

type GoldStagingRow struct {
	Earner     string
	Snapshot   string
	RewardHash string
	Token      string
	Amount     string
}

func (rc *RewardsCalculator) ListGoldStagingRowsForSnapshot(snapshotDate string) ([]*GoldStagingRow, error) {
	results := make([]*GoldStagingRow, 0)
	query := `
	SELECT
		earner,
		snapshot::text as snapshot,
		reward_hash,
		token,
		amount
	FROM gold_7_staging WHERE DATE(snapshot) < @cutoffDate`
	res := rc.grm.Raw(query, sql.Named("cutoffDate", snapshotDate)).Scan(&results)
	if res.Error != nil {
		rc.logger.Sugar().Errorw("Failed to list gold staging rows", "error", res.Error)
		return nil, res.Error
	}
	return results, nil
}
