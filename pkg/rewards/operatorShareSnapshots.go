package rewards

import (
	"database/sql"
)

const operatorShareSnapshotsQuery = `
with operator_shares_with_block_info as (
	select
		os.operator,
		os.strategy,
		os.shares,
		os.block_number,
		b.block_time,
		DATE(b.block_time) as block_date
	from operator_shares as os
	left join blocks as b on (b.number = os.block_number)
),
ranked_operator_records as (
    SELECT *,
           ROW_NUMBER() OVER (PARTITION BY operator, strategy, cast(block_time AS DATE) ORDER BY block_time DESC) AS rn
    FROM operator_shares_with_block_info
),
-- Get the latest record for each day & round up to the snapshot day
snapshotted_records as (
 SELECT
	 operator,
	 strategy,
	 shares,
	 block_time,
	 date_trunc('day', block_time) + INTERVAL '1' day as snapshot_time
 from ranked_operator_records
 where rn = 1
),
-- Get the range for each operator, strategy pairing
operator_share_windows as (
 SELECT
	 operator, strategy, shares, snapshot_time as start_time,
	 CASE
		 -- If the range does not have the end, use the current timestamp truncated to 0 UTC
		 WHEN LEAD(snapshot_time) OVER (PARTITION BY operator, strategy ORDER BY snapshot_time) is null THEN date_trunc('day', DATE(@cutoffDate))
		 ELSE LEAD(snapshot_time) OVER (PARTITION BY operator, strategy ORDER BY snapshot_time)
		 END AS end_time
 FROM snapshotted_records
),
cleaned_records as (
	SELECT * FROM operator_share_windows
	WHERE start_time < end_time
),
final_results as (
	SELECT
		operator,
		strategy,
		shares,
		cast(day AS DATE) AS snapshot
	FROM
		cleaned_records
			CROSS JOIN
		generate_series(DATE(start_time), DATE(end_time) - interval '1' day, interval '1' day) AS day
)
select * from final_results
where
	snapshot >= @startDate
	and snapshot < @cutoffDate
`

func (r *RewardsCalculator) GenerateOperatorShareSnapshots(startDate string, snapshotDate string) ([]*OperatorShareSnapshots, error) {
	results := make([]*OperatorShareSnapshots, 0)

	res := r.grm.Raw(operatorShareSnapshotsQuery,
		sql.Named("startDate", startDate),
		sql.Named("cutoffDate", snapshotDate),
	).Scan(&results)

	if res.Error != nil {
		r.logger.Sugar().Errorw("Failed to generate operator share snapshots", "error", res.Error)
		return nil, res.Error
	}
	return results, nil
}

func (r *RewardsCalculator) GenerateAndInsertOperatorShareSnapshots(startDate string, snapshotDate string) error {
	snapshots, err := r.GenerateOperatorShareSnapshots(startDate, snapshotDate)
	if err != nil {
		r.logger.Sugar().Errorw("Failed to generate operator share snapshots", "error", err)
		return err
	}

	r.logger.Sugar().Infow("Inserting operator share snapshots", "count", len(snapshots))
	res := r.grm.Model(&OperatorShareSnapshots{}).CreateInBatches(snapshots, 100)
	if res.Error != nil {
		r.logger.Sugar().Errorw("Failed to insert operator share snapshots", "error", res.Error)
		return res.Error
	}

	return nil
}
