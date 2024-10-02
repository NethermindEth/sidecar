package rewards

const _8_goldFinalQuery = `
insert into gold_table
SELECT
    earner,
    snapshot,
    reward_hash,
    token,
    amount
FROM gold_7_staging
`

func (rc *RewardsCalculator) GenerateGold8FinalTable() error {
	res := rc.grm.Exec(_8_goldFinalQuery)
	if res.Error != nil {
		rc.logger.Sugar().Errorw("Failed to create gold_final", "error", res.Error)
		return res.Error
	}
	return nil
}

func (rc *RewardsCalculator) Create8GoldTable() error {
	query := `
		create table if not exists gold_table (
			earner TEXT NOT NULL,
			snapshot DATE NOT NULL,
			reward_hash TEXT NOT NULL,
			token TEXT NOT NULL,
			amount TEXT NOT NULL
		)
	`
	res := rc.grm.Exec(query)
	if res.Error != nil {
		rc.logger.Sugar().Errorw("Failed to create gold_table", "error", res.Error)
		return res.Error
	}
	return nil
}
