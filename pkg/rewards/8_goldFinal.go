package rewards

import (
	"go.uber.org/zap"
)

const _8_goldFinalQuery = `
insert into gold_table
SELECT
    earner,
    snapshot,
    reward_hash,
    token,
    amount
FROM {{.goldStagingTable}}
`

func (rc *RewardsCalculator) GenerateGold8FinalTable(startDate string, snapshotDate string) error {
	allTableNames := getGoldTableNames(snapshotDate)

	rc.logger.Sugar().Infow("Generating rewards for all table",
		zap.String("startDate", startDate),
		zap.String("cutoffDate", snapshotDate),
	)

	query, err := renderQueryTemplate(_8_goldFinalQuery, map[string]string{
		"goldStagingTable": allTableNames[Table_7_GoldStaging],
	})
	if err != nil {
		rc.logger.Sugar().Errorw("Failed to render query template", "error", err)
		return err
	}

	res := rc.grm.Exec(query)
	if res.Error != nil {
		rc.logger.Sugar().Errorw("Failed to create gold_final", "error", res.Error)
		return res.Error
	}
	return nil
}
