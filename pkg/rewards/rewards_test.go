package rewards

import (
	"fmt"
	"github.com/Layr-Labs/go-sidecar/internal/config"
	"github.com/Layr-Labs/go-sidecar/internal/logger"
	"github.com/Layr-Labs/go-sidecar/internal/sqlite/migrations"
	"github.com/Layr-Labs/go-sidecar/internal/tests"
	"github.com/Layr-Labs/go-sidecar/internal/types/numbers"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

const TOTAL_BLOCK_COUNT = 1229187

func rewardsTestsEnabled() bool {
	return os.Getenv("TEST_REWARDS") == "true"
}

func getProjectRootPath() string {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	p, err := filepath.Abs(fmt.Sprintf("%s/../..", wd))
	if err != nil {
		panic(err)
	}
	return p
}

func hydrateAllBlocksTable(grm *gorm.DB, l *zap.Logger) error {
	projectRoot := getProjectRootPath()
	contents, err := tests.GetAllBlocksSqlFile(projectRoot)

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

func setupRewards() (
	string,
	*config.Config,
	*gorm.DB,
	*zap.Logger,
	error,
) {
	cfg := tests.GetConfig()
	cfg.Debug = true
	l, _ := logger.NewLogger(&logger.LoggerConfig{Debug: cfg.Debug})

	dbFileName, db, err := tests.GetFileBasedSqliteDatabaseConnection(l)
	if err != nil {
		panic(err)
	}
	sqliteMigrator := migrations.NewSqliteMigrator(db, l)
	if err := sqliteMigrator.MigrateAll(); err != nil {
		l.Sugar().Fatalw("Failed to migrate", "error", err)
	}

	return dbFileName, cfg, db, l, err
}

func teardownRewards(grm *gorm.DB) {
	teardownOperatorAvsRegistrationSnapshot(grm)
	teardownOperatorAvsStrategyWindows(grm)
	teardownOperatorShareSnapshot(grm)
	teardownStakerDelegationSnapshot(grm)
	teardownStakerShareSnapshot(grm)
}

func Test_Rewards(t *testing.T) {
	if !rewardsTestsEnabled() {
		t.Skipf("Skipping %s", t.Name())
		return
	}

	if err := numbers.InitPython(); err != nil {
		t.Fatal(err)
	}

	dbFileName, cfg, grm, l, err := setupRewards()
	fmt.Printf("Using db file: %+v\n", dbFileName)

	if err != nil {
		t.Fatal(err)
	}

	snapshotDate := "2024-09-01"

	t.Run("Should initialize the rewards calculator", func(t *testing.T) {
		rc, err := NewRewardsCalculator(l, grm, cfg)
		assert.Nil(t, err)
		if err != nil {
			t.Fatal(err)
		}
		assert.NotNil(t, rc)

		fmt.Printf("DB Path: %+v", dbFileName)

		query := `select name from main.sqlite_master where type = 'table' order by name asc`
		type row struct{ Name string }
		var tables []row
		res := rc.grm.Raw(query).Scan(&tables)
		assert.Nil(t, res.Error)

		expectedTables := []string{
			"combined_rewards",
			"gold_1_active_rewards",
			"gold_2_staker_reward_amounts",
			"gold_3_operator_reward_amounts",
			"gold_4_rewards_for_all",
			"gold_5_rfae_stakers",
			"gold_6_rfae_operators",
			"gold_7_staging",
			"gold_table",
			"operator_avs_registration_snapshots",
			"operator_avs_strategy_snapshots",
			"operator_share_snapshots",
			"staker_delegation_snapshots",
			"staker_share_snapshots",
		}
		tablesList := make([]string, 0)
		for i, table := range tables {
			fmt.Printf("[%v]: %+v\n", i, table.Name)
			tablesList = append(tablesList, table.Name)
		}

		for _, table := range expectedTables {
			assert.True(t, slices.Contains(tablesList, table))
		}

		// Setup all tables and source data
		err = hydrateAllBlocksTable(grm, l)
		assert.Nil(t, err)

		err = hydrateOperatorAvsStateChangesTable(grm, l)
		assert.Nil(t, err)

		err = hydrateOperatorAvsRestakedStrategies(grm, l)
		assert.Nil(t, err)

		err = hydrateOperatorShares(grm, l)
		assert.Nil(t, err)

		err = hydrateStakerDelegations(grm, l)
		assert.Nil(t, err)

		err = hydrateStakerShares(grm, l)
		assert.Nil(t, err)

		err = hydrateRewardSubmissionsTable(grm, l)
		assert.Nil(t, err)

		t.Log("Hydrated tables")

		// Generate snapshots
		err = rc.generateSnapshotData(snapshotDate)
		assert.Nil(t, err)

		t.Log("Generated and inserted snapshots")

		startDate := "1970-01-01"
		err = rc.GenerateActiveRewards(snapshotDate, startDate)
		assert.Nil(t, err)

		query = `select count(*) from gold_1_active_rewards`
		var count int
		res = rc.grm.Raw(query).Scan(&count)
		assert.Nil(t, res.Error)
		fmt.Printf("Count: %v\n", count)

		fmt.Printf("Done!\n\n")
		t.Cleanup(func() {
			teardownRewards(grm)
			// tests.DeleteTestSqliteDB(dbFileName)
		})
	})
}
