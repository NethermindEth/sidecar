package rewards

import "database/sql"

const _8_goldFinalQuery = `
insert into gold_table
SELECT
    earner,
    snapshot,
    reward_hash,
    token,
    amount
FROM gold_7_staging
where
	DATE(snapshot) >= @startDate
	and DATE(snapshot) < @cutoffDate
`

func (rc *RewardsCalculator) GenerateGold8FinalTable(startDate string, snapshotDate string) error {
	res := rc.grm.Exec(_8_goldFinalQuery,
		sql.Named("startDate", startDate),
		sql.Named("cutoffDate", snapshotDate),
	)
	if res.Error != nil {
		rc.logger.Sugar().Errorw("Failed to create gold_final", "error", res.Error)
		return res.Error
	}
	return nil
}
