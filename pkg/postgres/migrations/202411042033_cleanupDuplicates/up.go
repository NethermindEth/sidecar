package _202411042033_cleanupDuplicates

import (
	"database/sql"
	"github.com/Layr-Labs/go-sidecar/pkg/postgres/helpers"
	"gorm.io/gorm"
)

type Migration struct {
}

func (m *Migration) Up(db *sql.DB, grm *gorm.DB) error {
	queries := []string{
		`with ordered_rows as (
			select
				*,
				ROW_NUMBER() OVER (PARTITION BY reward_hash, strategy, strategy_index order by block_number asc) as rn
			from reward_submissions
		)
		delete from reward_submissions
		where (reward_hash, strategy, strategy_index, block_number) in (
			select o.reward_hash, o.strategy, o.strategy_index, o.block_number
			from ordered_rows as o
			where o.rn > 1
		)`,
		`with ordered_rows as (
			select
				*,
				ROW_NUMBER() OVER (PARTITION BY root_index order by block_number asc) as rn
			from submitted_distribution_roots
		)
		delete from submitted_distribution_roots
		where (root_index, block_number) in (
			select o.root_index, o.block_number
			from ordered_rows as o
			where o.rn > 1
		)`,
	}
	_, err := helpers.WrapTxAndCommit(func(tx *gorm.DB) (interface{}, error) {
		for _, query := range queries {
			res := tx.Exec(query)
			if res.Error != nil {
				return nil, res.Error
			}
		}
		return nil, nil
	}, grm, nil)
	return err
}

func (m *Migration) GetName() string {
	return "202411042033_cleanupDuplicates"
}
