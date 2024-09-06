package tests

import (
	"fmt"
	"github.com/Layr-Labs/sidecar/internal/config"
	"github.com/Layr-Labs/sidecar/internal/postgres"
	"github.com/Layr-Labs/sidecar/internal/tests"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
	"testing"
)

func setup() (
	*config.Config,
	*postgres.Postgres,
	*gorm.DB,
	error,
) {
	cfg := tests.GetConfig()

	pg, grm, err := tests.GetDatabaseConnection(cfg)

	return cfg, pg, grm, err
}

func TestDoesWrappedTransactionRollback(t *testing.T) {
	_, pg, grm, err := setup()
	if err != nil {
		t.Fatal(err)
	}
	defer pg.Db.Close()

	query := `create table test_table (id int);`
	_, err = pg.Db.Exec(query)
	if err != nil {
		t.Fatal(err)
	}

	_, err = postgres.WrapTxAndCommit[interface{}](func(tx *gorm.DB) (interface{}, error) {
		for i := 0; i < 10; i++ {
			query := "INSERT INTO test_table (id) VALUES (?)"
			res := tx.Exec(query, i)
			if res.Error != nil {
				return nil, err
			}
		}
		t.Logf("Inserted 10 rows, simulating a failure")
		return nil, fmt.Errorf("simulated failure")
	}, grm, nil)

	assert.NotNil(t, err)

	selectQuery := `select count(*) from test_table;`
	var count int
	err = pg.Db.QueryRow(selectQuery).Scan(&count)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Found '%d' rows", count)
	assert.Equal(t, 0, count)

	dropQuery := `drop table test_table;`
	_, err = pg.Db.Exec(dropQuery)
	if err != nil {
		t.Fatal(err)
	}

}

func TestDoesWrappedNestedTransactionRollback(t *testing.T) {
	_, pg, grm, err := setup()
	if err != nil {
		t.Fatal(err)
	}
	defer pg.Db.Close()

	query := `create table test_table (id int);`
	_, err = pg.Db.Exec(query)
	if err != nil {
		t.Fatal(err)
	}

	_, err = postgres.WrapTxAndCommit[interface{}](func(tx *gorm.DB) (interface{}, error) {
		_, err = postgres.WrapTxAndCommit[interface{}](func(tx *gorm.DB) (interface{}, error) {
			for i := 0; i < 10; i++ {
				query := "INSERT INTO test_table (id) VALUES (?)"
				res := tx.Exec(query, i)
				if res.Error != nil {
					return nil, err
				}
			}
			t.Logf("Inserted 10 rows")
			var count int
			tx.Raw(`select count(*) from test_table;`).Scan(&count)
			assert.Equal(t, 10, count)
			t.Logf("Found '%d' rows", count)
			return nil, nil
		}, grm, tx)
		_, err = postgres.WrapTxAndCommit[interface{}](func(tx *gorm.DB) (interface{}, error) {
			for i := 0; i < 10; i++ {
				query := "INSERT INTO test_table (id) VALUES (?)"
				res := tx.Exec(query, i)
				if res.Error != nil {
					return nil, err
				}
			}
			t.Logf("Inserted 10 rows")
			var count int
			tx.Raw(`select count(*) from test_table;`).Scan(&count)
			t.Logf("Found '%d' rows", count)
			assert.Equal(t, 20, count)
			t.Logf("Inserted 10 rows")
			return nil, nil
		}, grm, tx)
		return postgres.WrapTxAndCommit[interface{}](func(tx *gorm.DB) (interface{}, error) {
			for i := 0; i < 10; i++ {
				query := "INSERT INTO test_table (id) VALUES (?)"
				res := tx.Exec(query, i)
				if res.Error != nil {
					return nil, err
				}
			}
			t.Logf("Inserted 10 rows")
			var count int
			tx.Raw(`select count(*) from test_table;`).Scan(&count)
			t.Logf("Found '%d' rows", count)
			assert.Equal(t, 30, count)
			t.Logf("Inserted 10 rows, simulating a failure")
			return nil, fmt.Errorf("simulated failure")
		}, grm, tx)

	}, grm, nil)

	assert.NotNil(t, err)

	selectQuery := `select count(*) from test_table;`
	var count int
	err = pg.Db.QueryRow(selectQuery).Scan(&count)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Found '%d' rows", count)
	assert.Equal(t, 0, count)

	dropQuery := `drop table test_table;`
	_, err = pg.Db.Exec(dropQuery)
	if err != nil {
		t.Fatal(err)
	}

}
