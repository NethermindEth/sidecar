package stakerShares

import (
	"fmt"
	"github.com/Layr-Labs/go-sidecar/internal/tests"
	"github.com/Layr-Labs/go-sidecar/pkg/eigenState/stateManager"
	"github.com/Layr-Labs/go-sidecar/pkg/postgres"
	"github.com/Layr-Labs/go-sidecar/pkg/storage"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func getProjectRootPath() string {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	p, err := filepath.Abs(fmt.Sprintf("%s/../../..", wd))
	if err != nil {
		panic(err)
	}
	return p
}

func hydrateAllBlocksTable(grm *gorm.DB, l *zap.Logger) (int, error) {
	projectRoot := getProjectRootPath()
	contents, err := tests.GetAllBlocksSqlFile(projectRoot)

	if err != nil {
		return 0, err
	}

	count := len(strings.Split(strings.Trim(contents, "\n"), "\n")) - 1

	res := grm.Exec(contents)
	if res.Error != nil {
		l.Sugar().Errorw("Failed to execute sql", "error", zap.Error(res.Error))
		return count, res.Error
	}
	return count, nil
}

func hydrateStakerShareTransactionLogs(grm *gorm.DB, l *zap.Logger) error {
	projectRoot := getProjectRootPath()
	contents, err := tests.GetStakerSharesTransactionLogsSqlFile(projectRoot)

	if err != nil {
		return err
	}

	res := grm.Exec(contents)
	if res.Error != nil {
		l.Sugar().Errorw("Failed to execute sql", "error", zap.Error(res.Error))
		return res.Error
	}
	return nil
}

type blockRange struct {
	minBlockNumber uint64
	maxBlockNumber uint64
}

func Test_StakerSharesIntegration(t *testing.T) {
	dbName, grm, l, cfg, err := setup()

	if err != nil {
		t.Fatal(err)
	}

	t.Run("Staker shares calculation through M1 and M2 migration", func(t *testing.T) {
		esm := stateManager.NewEigenStateManager(l, grm)
		model, err := NewStakerSharesModel(esm, grm, l, cfg)
		assert.Nil(t, err)

		_, err = hydrateAllBlocksTable(grm, l)
		assert.Nil(t, err)
		t.Logf("Hydrated all blocks table")

		err = hydrateStakerShareTransactionLogs(grm, l)
		assert.Nil(t, err)
		t.Logf("Hydrated staker share transaction logs")

		// get min/max block number
		var minMaxBlocks blockRange
		query := `select min(block_number) as min_block_number, max(block_number) as max_block_number from transaction_logs`
		res := grm.Raw(query).Scan(&minMaxBlocks)
		if res.Error != nil {
			t.Fatal(res.Error)
		}
		t.Logf("min block number: %d, max block number: %d", minMaxBlocks.minBlockNumber, minMaxBlocks.maxBlockNumber)

		for i := minMaxBlocks.minBlockNumber; i <= minMaxBlocks.maxBlockNumber; i++ {
			if err = model.SetupStateForBlock(i); err != nil {
				t.Logf("Failed to setup state for block %d", i)
				t.Fatal(err)
			}

			var logs []*storage.TransactionLog
			query := `select * from transaction_logs where block_number = ? order by log_index asc`
			res := grm.Raw(query, i).Scan(&logs)
			if res.Error != nil {
				t.Logf("Failed to get transaction logs for block %d", i)
				t.Fatal(res.Error)
			}

			for _, log := range logs {
				if _, err = model.HandleStateChange(log); err != nil {
					t.Logf("Failed to handle state change for block %d", i)
					t.Fatal(err)
				}
			}

			if err := model.CommitFinalState(i); err != nil {
				t.Logf("Failed to commit final state for block %d", i)
				t.Fatal(err)
			}

			if err := model.CleanupProcessedStateForBlock(i); err != nil {
				t.Logf("Failed to cleanup processed state for block %d", i)
				t.Fatal(err)
			}
		}

		var count int
		query = `select count(*) from staker_share_deltas`
		res = grm.Raw(query).Scan(&count)
		if res.Error != nil {
			t.Fatal(res.Error)
		}

		fmt.Printf("Total staker share deltas: %d\n", count)
	})

	t.Cleanup(func() {
		postgres.TeardownTestDatabase(dbName, cfg, grm, l)
	})
}
