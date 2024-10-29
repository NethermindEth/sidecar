package rewards

import (
	"fmt"
	"github.com/Layr-Labs/go-sidecar/internal/config"
	"github.com/Layr-Labs/go-sidecar/internal/logger"
	"github.com/Layr-Labs/go-sidecar/internal/postgres"
	"github.com/Layr-Labs/go-sidecar/internal/tests"
	"github.com/Layr-Labs/go-sidecar/pkg/utils"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// const TOTAL_BLOCK_COUNT = 1229187

func rewardsTestsEnabled() bool {
	return os.Getenv("TEST_REWARDS") == "true"
}

func getRewardsTestContext() string {
	ctx := os.Getenv("REWARDS_TEST_CONTEXT")
	if ctx == "" {
		return "testnet"
	}
	return ctx
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

func getSnapshotDate() (string, error) {
	context := getRewardsTestContext()

	switch context {
	case "testnet":
		return "2024-09-01", nil
	case "testnet-reduced":
		return "2024-07-25", nil
	case "mainnet-reduced":
		return "2024-08-12", nil
	}
	return "", fmt.Errorf("Unknown context: %s", context)
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

func getRowCountForTable(grm *gorm.DB, tableName string) (int, error) {
	query := fmt.Sprintf("select count(*) as cnt from %s", tableName)
	var count int
	res := grm.Raw(query).Scan(&count)

	if res.Error != nil {
		return 0, res.Error
	}
	return count, nil
}

func setupRewards() (
	string,
	*config.Config,
	*gorm.DB,
	*zap.Logger,
	error,
) {
	cfg := tests.GetConfig()
	cfg.Chain = config.Chain_Mainnet

	cfg.DatabaseConfig = *tests.GetDbConfigFromEnv()

	l, _ := logger.NewLogger(&logger.LoggerConfig{Debug: cfg.Debug})

	dbname, _, grm, err := postgres.GetTestPostgresDatabase(cfg.DatabaseConfig, l)
	if err != nil {
		return dbname, nil, nil, nil, err
	}

	return dbname, cfg, grm, l, nil
}

func teardownRewards(dbname string, cfg *config.Config, db *gorm.DB, l *zap.Logger) {
	rawDb, _ := db.DB()
	_ = rawDb.Close()

	pgConfig := postgres.PostgresConfigFromDbConfig(&cfg.DatabaseConfig)

	if err := postgres.DeleteTestDatabase(pgConfig, dbname); err != nil {
		l.Sugar().Errorw("Failed to delete test database", "error", err)
	}
}

func generateDateRange(startStr, endStr string) ([]string, error) {
	start, err := time.Parse(time.DateOnly, startStr)
	if err != nil {
		return nil, fmt.Errorf("invalid start date format: %v", err)
	}

	end, err := time.Parse(time.DateOnly, endStr)
	if err != nil {
		return nil, fmt.Errorf("invalid end date format: %v", err)
	}

	if start.After(end) {
		return []string{}, nil
	}

	days := int(end.Sub(start).Hours()/24) + 1
	dates := make([]string, days)

	for i := 0; i < days; i++ {
		dates[i] = start.AddDate(0, 0, i).Format(time.DateOnly)
	}

	return dates, nil
}

func Test_Rewards(t *testing.T) {
	if !rewardsTestsEnabled() {
		t.Skipf("Skipping %s", t.Name())
		return
	}

	dbFileName, cfg, grm, l, err := setupRewards()
	fmt.Printf("Using db file: %+v\n", dbFileName)

	if err != nil {
		t.Fatal(err)
	}

	projectRoot := getProjectRootPath()

	// snapshotDate, err := getSnapshotDate()
	// if err != nil {
	// 	t.Fatal(err)
	// }

	t.Run("Should initialize the rewards calculator", func(t *testing.T) {
		rc, err := NewRewardsCalculator(l, grm, cfg)
		assert.Nil(t, err)
		if err != nil {
			t.Fatal(err)
		}
		assert.NotNil(t, rc)

		fmt.Printf("DB Path: %+v\n", dbFileName)

		testStart := time.Now()

		// Setup all tables and source data
		_, err = hydrateAllBlocksTable(grm, l)
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

		// Each snapshot date present is the cutoff date - 1 day
		// 0. August 1, 2024 --> August 2, 2024
		// 1. August 11, 2024 --> August 12, 2024
		snapshotDates, err := generateDateRange("2024-08-02", "2024-08-10")
		if err != nil {
			t.Fatal(err)
		}

		snapshotDates = []string{"2024-08-02", "2024-08-11", "2024-08-12"}

		fmt.Printf("Hydration duration: %v\n", time.Since(testStart))
		testStart = time.Now()

		for i, snapshotDate := range snapshotDates {
			t.Log("-----------------------------\n")
			var startDate string
			if i == 0 {
				startDate = "1970-01-01"
			} else {
				startDate, err = rc.GetNextSnapshotDate()
				if err != nil {
					t.Fatal(err)
				}
				t.Logf("Max snapshot date: %s", startDate)
			}

			snapshotStartTime := time.Now()

			t.Logf("Generating rewards - startDate %s, snapshotDate: %s", startDate, snapshotDate)
			// Generate snapshots
			err = rc.generateSnapshotData(startDate, snapshotDate)
			assert.Nil(t, err)

			goldTableNames := getGoldTableNames(snapshotDate)

			fmt.Printf("Snapshot duration: %v\n", time.Since(testStart))
			testStart = time.Now()

			t.Log("Generated and inserted snapshots")
			forks, err := cfg.GetForkDates()
			assert.Nil(t, err)

			fmt.Printf("Running gold_1_active_rewards\n")
			err = rc.Generate1ActiveRewards(startDate, snapshotDate)
			assert.Nil(t, err)
			rows, err := getRowCountForTable(grm, goldTableNames[Table_1_ActiveRewards])
			assert.Nil(t, err)
			fmt.Printf("\tRows in gold_1_active_rewards: %v - [time: %v]\n", rows, time.Since(testStart))
			testStart = time.Now()

			fmt.Printf("Running gold_2_staker_reward_amounts %+v\n", time.Now())
			err = rc.GenerateGold2StakerRewardAmountsTable(startDate, snapshotDate, forks)
			assert.Nil(t, err)
			rows, err = getRowCountForTable(grm, goldTableNames[Table_2_StakerRewardAmounts])
			assert.Nil(t, err)
			fmt.Printf("\tRows in gold_2_staker_reward_amounts: %v - [time: %v]\n", rows, time.Since(testStart))
			testStart = time.Now()

			fmt.Printf("Running gold_3_operator_reward_amounts\n")
			err = rc.GenerateGold3OperatorRewardAmountsTable(startDate, snapshotDate)
			assert.Nil(t, err)
			rows, err = getRowCountForTable(grm, goldTableNames[Table_3_OperatorRewardAmounts])
			assert.Nil(t, err)
			fmt.Printf("\tRows in gold_3_operator_reward_amounts: %v - [time: %v]\n", rows, time.Since(testStart))
			testStart = time.Now()

			fmt.Printf("Running gold_4_rewards_for_all\n")
			err = rc.GenerateGold4RewardsForAllTable(startDate, snapshotDate)
			assert.Nil(t, err)
			rows, err = getRowCountForTable(grm, goldTableNames[Table_4_RewardsForAll])
			assert.Nil(t, err)
			fmt.Printf("\tRows in gold_4_rewards_for_all: %v - [time: %v]\n", rows, time.Since(testStart))
			testStart = time.Now()

			fmt.Printf("Running gold_5_rfae_stakers\n")
			err = rc.GenerateGold5RfaeStakersTable(startDate, snapshotDate, forks)
			assert.Nil(t, err)
			rows, err = getRowCountForTable(grm, goldTableNames[Table_5_RfaeStakers])
			assert.Nil(t, err)
			fmt.Printf("\tRows in gold_5_rfae_stakers: %v - [time: %v]\n", rows, time.Since(testStart))
			testStart = time.Now()

			fmt.Printf("Running gold_6_rfae_operators\n")
			err = rc.GenerateGold6RfaeOperatorsTable(startDate, snapshotDate)
			assert.Nil(t, err)
			rows, err = getRowCountForTable(grm, goldTableNames[Table_6_RfaeOperators])
			assert.Nil(t, err)
			fmt.Printf("\tRows in gold_6_rfae_operators: %v - [time: %v]\n", rows, time.Since(testStart))
			testStart = time.Now()

			fmt.Printf("Running gold_7_staging\n")
			err = rc.GenerateGold7StagingTable(startDate, snapshotDate)
			assert.Nil(t, err)
			rows, err = getRowCountForTable(grm, goldTableNames[Table_7_GoldStaging])
			assert.Nil(t, err)
			fmt.Printf("\tRows in gold_7_staging: %v - [time: %v]\n", rows, time.Since(testStart))
			testStart = time.Now()

			fmt.Printf("Running gold_8_final_table\n")
			err = rc.GenerateGold8FinalTable(startDate, snapshotDate)
			assert.Nil(t, err)
			rows, err = getRowCountForTable(grm, "gold_table")
			assert.Nil(t, err)
			fmt.Printf("\tRows in gold_table: %v - [time: %v]\n", rows, time.Since(testStart))
			testStart = time.Now()

			goldStagingRows, err := rc.ListGoldStagingRowsForSnapshot(snapshotDate)
			assert.Nil(t, err)

			t.Logf("Gold staging rows for snapshot %s: %d", snapshotDate, len(goldStagingRows))

			fmt.Printf("Total duration for rewards compute %s: %v\n", snapshotDate, time.Since(snapshotStartTime))
			testStart = time.Now()

			//if !slices.Contains([]string{"2024-08-02", "2024-08-12"}, snapshotDate) {
			//	t.Logf("Skipping gold staging validation for snapshot date: %s", snapshotDate)
			//	continue
			//}

			expectedRows, err := tests.GetGoldStagingExpectedResults(projectRoot, snapshotDate)
			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, len(expectedRows), len(goldStagingRows))
			t.Logf("Expected rows: %d, Gold staging rows: %d", len(expectedRows), len(goldStagingRows))

			missingRows := 0
			invalidAmounts := 0
			for i, row := range goldStagingRows {
				foundRow := utils.Find(expectedRows, func(r *tests.GoldStagingExpectedResult) bool {
					return row.Earner == r.Earner && row.Snapshot == r.Snapshot && row.RewardHash == r.RewardHash && row.Token == r.Token
				})
				if foundRow == nil {
					missingRows++
					if missingRows < 100 {
						fmt.Printf("[%d] Row not found in expected results: %+v\n", i, row)
					}
					continue
				}
				if foundRow.Amount != row.Amount {
					invalidAmounts++
					if invalidAmounts < 100 {
						fmt.Printf("[%d] Amount mismatch: expected '%s', got '%s' for row: %+v\n", i, foundRow.Amount, row.Amount, row)
					}
				}
			}
			assert.Zero(t, missingRows)
			t.Logf("Missing rows: %d", missingRows)

			assert.Zero(t, invalidAmounts)
			t.Logf("Invalid amounts: %d", invalidAmounts)
		}

		fmt.Printf("Done!\n\n")
		t.Cleanup(func() {
			// teardownRewards(dbFileName, cfg, grm, l)
		})
	})
}
