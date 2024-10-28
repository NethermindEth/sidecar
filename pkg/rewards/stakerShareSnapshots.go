package rewards

const stakerShareSnapshotsQuery = `
with staker_shares_with_block_info as (
	select
		ss.staker,
		ss.strategy,
		ss.shares,
		ss.block_number,
		b.block_time,
		DATE(b.block_time) as block_date
	from staker_shares as ss
	left join blocks as b on (b.number = ss.block_number)
),
ranked_staker_records as (
    SELECT *,
           ROW_NUMBER() OVER (PARTITION BY staker, strategy, cast(block_time AS DATE) ORDER BY block_time DESC) AS rn
    FROM staker_shares_with_block_info
),
-- Get the latest record for each day & round up to the snapshot day
snapshotted_records as (
 SELECT
	 staker,
	 strategy,
	 shares,
	 block_time,
	 date_trunc('day', block_time) + INTERVAL '1' day AS snapshot_time
 from ranked_staker_records
 where rn = 1
),
-- Get the range for each operator, strategy pairing
staker_share_windows as (
 SELECT
	 staker, strategy, shares, snapshot_time as start_time,
	 CASE
		 -- If the range does not have the end, use the current timestamp truncated to 0 UTC
		 WHEN LEAD(snapshot_time) OVER (PARTITION BY staker, strategy ORDER BY snapshot_time) is null THEN date_trunc('day', TIMESTAMP '{{.cutoffDate}}')
		 ELSE LEAD(snapshot_time) OVER (PARTITION BY staker, strategy ORDER BY snapshot_time)
		 END AS end_time
 FROM snapshotted_records
),
cleaned_records as (
	SELECT * FROM staker_share_windows
	WHERE start_time < end_time
),
final_results as (
	SELECT
		staker,
		strategy,
		shares,
		cast(day AS DATE) AS snapshot
	FROM
		cleaned_records
	CROSS JOIN
		generate_series(DATE(start_time), DATE(end_time) - interval '1' day, interval '1' day) AS day
)
select * from final_results
`

func (r *RewardsCalculator) GenerateStakerShareSnapshots(startDate string, snapshotDate string) ([]*StakerShareSnapshot, error) {
	results := make([]*StakerShareSnapshot, 0)

	query, err := renderQueryTemplate(stakerShareSnapshotsQuery, map[string]string{
		"cutoffDate": snapshotDate,
	})
	if err != nil {
		r.logger.Sugar().Errorw("Failed to render query template", "error", err)
		return nil, err
	}

	res := r.grm.Raw(query).Scan(&results)

	if res.Error != nil {
		r.logger.Sugar().Errorw("Failed to generate staker share snapshots", "error", res.Error)
		return nil, res.Error
	}
	return results, nil
}

func (r *RewardsCalculator) GenerateAndInsertStakerShareSnapshots(startDate string, snapshotDate string) error {
	tableName := "staker_share_snapshots"

	query, err := renderQueryTemplate(stakerShareSnapshotsQuery, map[string]string{
		"cutoffDate": snapshotDate,
	})
	if err != nil {
		r.logger.Sugar().Errorw("Failed to render query template", "error", err)
		return err
	}

	err = r.generateAndInsertFromQuery(tableName, query, nil)
	if err != nil {
		r.logger.Sugar().Errorw("Failed to generate staker_share_snapshots", "error", err)
		return err
	}
	return nil
}
