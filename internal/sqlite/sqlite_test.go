package sqlite

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_Sqlite(t *testing.T) {
	// t.Run("Should create a new GormSqlite", func(t *testing.T) {
	// 	s := NewSqlite("file::memory:?cache=shared")
	// 	grm, err := NewGormSqliteFromSqlite(s)
	// 	assert.Nil(t, err)
	// 	assert.NotNil(t, grm)
	//
	// 	db, err := grm.DB()
	// 	defer db.Close()
	// })
	t.Run("Should use the bytesToHex function", func(t *testing.T) {
		query := `
			with json_values as (
				select 
					cast('{"newWithdrawalRoot": [218, 200, 138, 86, 38, 9, 156, 119, 73, 13, 168, 40, 209, 43, 238, 83, 234, 177, 230, 73, 120, 205, 255, 143, 255, 216, 51, 209, 137, 100, 163, 233] }' as text) as json_col
				from (select 1)
			)
			select
				bytes_to_hex(json_extract(json_col, '$.newWithdrawalRoot')) AS withdrawal_hex
			from json_values
			limit 1
		`
		s := NewSqlite("file::memory:?cache=shared")
		grm, err := NewGormSqliteFromSqlite(s)
		assert.Nil(t, err)

		type results struct {
			WithdrawalHex string
		}

		hexValue := &results{}
		res := grm.Raw(query).Scan(&hexValue)
		assert.Nil(t, res.Error)
		assert.Equal(t, "dac88a5626099c77490da828d12be653eab1e64978cdff8fffD337d18964a3e9", hexValue.WithdrawalHex)
	})
}
