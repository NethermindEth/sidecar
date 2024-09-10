package rewardSubmissions

import (
	"github.com/Layr-Labs/go-sidecar/internal/config"
	"github.com/Layr-Labs/go-sidecar/internal/eigenState/stateManager"
	"github.com/Layr-Labs/go-sidecar/internal/logger"
	"github.com/Layr-Labs/go-sidecar/internal/sqlite/migrations"
	"github.com/Layr-Labs/go-sidecar/internal/storage"
	"github.com/Layr-Labs/go-sidecar/internal/tests"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"math/big"
	"strings"
	"testing"
	"time"
)

func setup() (
	*config.Config,
	*gorm.DB,
	*zap.Logger,
	error,
) {
	cfg := tests.GetConfig()
	l, _ := logger.NewLogger(&logger.LoggerConfig{Debug: cfg.Debug})

	db, err := tests.GetSqliteDatabaseConnection()
	if err != nil {
		panic(err)
	}
	sqliteMigrator := migrations.NewSqliteMigrator(db, l)
	if err := sqliteMigrator.MigrateAll(); err != nil {
		l.Sugar().Fatalw("Failed to migrate", "error", err)
	}

	return cfg, db, l, err
}

func teardown(model *RewardSubmissionsModel) {
	queries := []string{
		`truncate table reward_submissions cascade`,
	}
	for _, query := range queries {
		model.Db.Raw(query)
	}
}

func Test_RewardSubmissions(t *testing.T) {
	cfg, grm, l, err := setup()

	if err != nil {
		t.Fatal(err)
	}

	esm := stateManager.NewEigenStateManager(l, grm)

	model, err := NewRewardSubmissionsModel(esm, grm, cfg.Network, cfg.Environment, l, cfg)

	t.Run("Handle a range payment submission", func(t *testing.T) {
		blockNumber := uint64(100)

		block := &storage.Block{
			Number:    blockNumber,
			Hash:      "some hash",
			BlockTime: time.Now().Add(time.Hour * 100),
		}
		res := model.Db.Model(&storage.Block{}).Create(block)
		if res.Error != nil {
			t.Fatal(res.Error)
		}

		log := &storage.TransactionLog{
			TransactionHash:  "some hash",
			TransactionIndex: big.NewInt(100).Uint64(),
			BlockNumber:      blockNumber,
			Address:          cfg.GetContractsMapForEnvAndNetwork().RewardsCoordinator,
			Arguments:        `[{"Name": "avs", "Type": "address", "Value": "0x00526A07855f743964F05CccAeCcf7a9E34847fF"}, {"Name": "paymentNonce", "Type": "uint256", "Value": "0x0000000000000000000000000000000000000000"}, {"Name": "rangePaymentHash", "Type": "bytes32", "Value": "0x58959fBe6661daEA647E20dF7c6d2c7F0d2215fB"}, {"Name": "rangePayment", "Type": "((address,uint96)[],address,uint256,uint32,uint32)", "Value": ""}]`,
			EventName:        "RangePaymentCreated",
			LogIndex:         big.NewInt(12).Uint64(),
			OutputData:       `{"rangePayment": {"token": "0x94373a4919b3240d86ea41593d5eba789fef3848", "amount": 50000000000000000000, "duration": 2419200, "startTimestamp": 1712188800, "strategiesAndMultipliers": [{"strategy": "0x3c28437e610fb099cc3d6de4d9c707dfacd308ae", "multiplier": 1000000000000000000}, {"strategy": "0x3cb1fd19cfb178c1098f2fc1e11090a0642b2314", "multiplier": 2000000000000000000}, {"strategy": "0x5c8b55722f421556a2aafb7a3ea63d4c3e514312", "multiplier": 3000000000000000000}, {"strategy": "0x6dc6ce589f852f96ac86cb160ab0b15b9f56dedd", "multiplier": 4500000000000000000}, {"strategy": "0x87f6c7d24b109919eb38295e3f8298425e6331d9", "multiplier": 500000000000000000}, {"strategy": "0xd523267698c81a372191136e477fdebfa33d9fb4", "multiplier": 8000000000000000000}, {"strategy": "0xdccf401fd121d8c542e96bc1d0078884422afad2", "multiplier": 5000000000000000000}]}}`,
		}

		err = model.InitBlockProcessing(blockNumber)
		assert.Nil(t, err)

		isInteresting := model.IsInterestingLog(log)
		assert.True(t, isInteresting)

		change, err := model.HandleStateChange(log)
		assert.Nil(t, err)
		assert.NotNil(t, change)

		strategiesAndMultipliers := []struct {
			Strategy   string
			Multiplier string
		}{
			{"0x3c28437e610fb099cc3d6de4d9c707dfacd308ae", "1000000000000000000"},
			{"0x3cb1fd19cfb178c1098f2fc1e11090a0642b2314", "2000000000000000000"},
			{"0x5c8b55722f421556a2aafb7a3ea63d4c3e514312", "3000000000000000000"},
			{"0x6dc6ce589f852f96ac86cb160ab0b15b9f56dedd", "4500000000000000000"},
			{"0x87f6c7d24b109919eb38295e3f8298425e6331d9", "500000000000000000"},
			{"0xd523267698c81a372191136e477fdebfa33d9fb4", "8000000000000000000"},
			{"0xdccf401fd121d8c542e96bc1d0078884422afad2", "5000000000000000000"},
		}

		typedChange := change.(*RewardSubmissions)
		assert.Equal(t, len(strategiesAndMultipliers), len(typedChange.Submissions))

		for i, submission := range typedChange.Submissions {
			assert.Equal(t, strings.ToLower("0x00526A07855f743964F05CccAeCcf7a9E34847fF"), strings.ToLower(submission.Avs))
			assert.Equal(t, strings.ToLower("0x94373a4919b3240d86ea41593d5eba789fef3848"), strings.ToLower(submission.Token))
			assert.Equal(t, strings.ToLower("0x58959fBe6661daEA647E20dF7c6d2c7F0d2215fB"), strings.ToLower(submission.RewardHash))
			assert.Equal(t, "50000000000000000000", submission.Amount)
			assert.Equal(t, uint64(2419200), submission.Duration)
			assert.Equal(t, int64(1712188800), submission.StartTimestamp.Unix())
			assert.Equal(t, int64(2419200+1712188800), submission.EndTimestamp.Unix())

			assert.Equal(t, strings.ToLower(strategiesAndMultipliers[i].Strategy), strings.ToLower(submission.Strategy))
			assert.Equal(t, strategiesAndMultipliers[i].Multiplier, submission.Multiplier)
		}

		err = model.CommitFinalState(blockNumber)
		assert.Nil(t, err)

		query := `select count(*) from reward_submissions`
		var count int
		res = model.Db.Raw(query, blockNumber).Scan(&count)
		assert.Nil(t, res.Error)
		assert.Equal(t, len(strategiesAndMultipliers), count)

		stateRoot, err := model.GenerateStateRoot(blockNumber)
		assert.Nil(t, err)
		assert.NotNil(t, stateRoot)
		assert.True(t, len(stateRoot) > 0)

		teardown(model)
	})

	t.Run("Handle a range payment for all submission", func(t *testing.T) {

	})

	t.Run("Handle a reward submission", func(t *testing.T) {

	})

	t.Run("Handle a reward submission for all", func(t *testing.T) {

	})
}
