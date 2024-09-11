package rewardSubmissions

import (
	"fmt"
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
		`truncate table blocks cascade`,
	}
	for _, query := range queries {
		res := model.Db.Raw(query)
		if res.Error != nil {
			fmt.Printf("Failed to run query: %v\n", res.Error)
		}
	}
}

func Test_RewardSubmissions(t *testing.T) {
	cfg, grm, l, err := setup()

	if err != nil {
		t.Fatal(err)
	}

	esm := stateManager.NewEigenStateManager(l, grm)

	model, err := NewRewardSubmissionsModel(esm, grm, cfg.Network, cfg.Environment, l, cfg)

	submissionCounter := 0

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

		rewards := make([]*RewardSubmission, 0)
		query := `select * from reward_submissions where block_number = ?`
		res = model.Db.Raw(query, blockNumber).Scan(&rewards)
		assert.Nil(t, res.Error)
		assert.Equal(t, len(strategiesAndMultipliers), len(rewards))

		submissionCounter += len(rewards)

		stateRoot, err := model.GenerateStateRoot(blockNumber)
		assert.Nil(t, err)
		assert.NotNil(t, stateRoot)
		assert.True(t, len(stateRoot) > 0)
	})

	t.Run("Handle a range payment for all submission", func(t *testing.T) {
		blockNumber := uint64(101)

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
			Arguments:        `[{"Name": "submitter", "Type": "address", "Value": "0x00526A07855f743964F05CccAeCcf7a9E34847fF"}, {"Name": "paymentNonce", "Type": "uint256", "Value": "0x0000000000000000000000000000000000000001"}, {"Name": "rangePaymentHash", "Type": "bytes32", "Value": "0x69193C881C4BfA9015F1E9B2631e31238BedB93e"}, {"Name": "rangePayment", "Type": "((address,uint96)[],address,uint256,uint32,uint32)", "Value": ""}]`,
			EventName:        "RangePaymentForAllCreated",
			LogIndex:         big.NewInt(12).Uint64(),
			OutputData:       `{"rangePayment": {"token": "0x3f1c547b21f65e10480de3ad8e19faac46c95034", "amount": 11000000000000000000, "duration": 2419200, "startTimestamp": 1713398400, "strategiesAndMultipliers": [{"strategy": "0x5c8b55722f421556a2aafb7a3ea63d4c3e514312", "multiplier": 1000000000000000000}, {"strategy": "0x7fa77c321bf66e42eabc9b10129304f7f90c5585", "multiplier": 2000000000000000000}, {"strategy": "0xbeac0eeeeeeeeeeeeeeeeeeeeeeeeeeeeeebeac0", "multiplier": 3000000000000000000}, {"strategy": "0xd523267698c81a372191136e477fdebfa33d9fb4", "multiplier": 4500000000000000000}]}}`,
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
			{"0x5c8b55722f421556a2aafb7a3ea63d4c3e514312", "1000000000000000000"},
			{"0x7fa77c321bf66e42eabc9b10129304f7f90c5585", "2000000000000000000"},
			{"0xbeac0eeeeeeeeeeeeeeeeeeeeeeeeeeeeeebeac0", "3000000000000000000"},
			{"0xd523267698c81a372191136e477fdebfa33d9fb4", "4500000000000000000"},
		}

		typedChange := change.(*RewardSubmissions)
		assert.Equal(t, len(strategiesAndMultipliers), len(typedChange.Submissions))

		for i, submission := range typedChange.Submissions {
			assert.Equal(t, strings.ToLower("0x00526A07855f743964F05CccAeCcf7a9E34847fF"), strings.ToLower(submission.Avs))
			assert.Equal(t, strings.ToLower("0x3f1c547b21f65e10480de3ad8e19faac46c95034"), strings.ToLower(submission.Token))
			assert.Equal(t, strings.ToLower("0x69193C881C4BfA9015F1E9B2631e31238BedB93e"), strings.ToLower(submission.RewardHash))
			assert.Equal(t, "11000000000000000000", submission.Amount)
			assert.Equal(t, uint64(2419200), submission.Duration)
			assert.Equal(t, int64(1713398400), submission.StartTimestamp.Unix())
			assert.Equal(t, int64(2419200+1713398400), submission.EndTimestamp.Unix())

			assert.Equal(t, strings.ToLower(strategiesAndMultipliers[i].Strategy), strings.ToLower(submission.Strategy))
			assert.Equal(t, strategiesAndMultipliers[i].Multiplier, submission.Multiplier)

			fmt.Printf("Submission: %+v\n", submission)
		}

		err = model.CommitFinalState(blockNumber)
		assert.Nil(t, err)

		rewards := make([]*RewardSubmission, 0)
		query := `select * from reward_submissions where block_number = ?`
		res = model.Db.Raw(query, blockNumber).Scan(&rewards)
		assert.Nil(t, res.Error)
		assert.Equal(t, len(strategiesAndMultipliers), len(rewards))

		submissionCounter += len(strategiesAndMultipliers)

		stateRoot, err := model.GenerateStateRoot(blockNumber)
		assert.Nil(t, err)
		assert.NotNil(t, stateRoot)
		assert.True(t, len(stateRoot) > 0)

		teardown(model)
	})

	t.Run("Handle a reward submission", func(t *testing.T) {
		blockNumber := uint64(102)

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
			Arguments:        `[{"Name": "avs", "Type": "address", "Value": "0xd36b6e5eee8311d7bffb2f3bb33301a1ab7de101", "Indexed": true}, {"Name": "submissionNonce", "Type": "uint256", "Value": 0, "Indexed": true}, {"Name": "rewardsSubmissionHash", "Type": "bytes32", "Value": "0x7402669fb2c8a0cfe8108acb8a0070257c77ec6906ecb07d97c38e8a5ddc66a9", "Indexed": true}, {"Name": "rewardsSubmission", "Type": "((address,uint96)[],address,uint256,uint32,uint32)", "Value": null, "Indexed": false}]`,
			EventName:        "AVSRewardsSubmissionCreated",
			LogIndex:         big.NewInt(12).Uint64(),
			OutputData:       `{"rewardsSubmission": {"token": "0x0ddd9dc88e638aef6a8e42d0c98aaa6a48a98d24", "amount": 10000000000000000000000, "duration": 2419200, "startTimestamp": 1725494400, "strategiesAndMultipliers": [{"strategy": "0x5074dfd18e9498d9e006fb8d4f3fecdc9af90a2c", "multiplier": 1000000000000000000}]}}`,
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
			{"0x5074dfd18e9498d9e006fb8d4f3fecdc9af90a2c", "1000000000000000000"},
		}

		typedChange := change.(*RewardSubmissions)
		assert.Equal(t, len(strategiesAndMultipliers), len(typedChange.Submissions))

		for i, submission := range typedChange.Submissions {
			assert.Equal(t, strings.ToLower("0xd36b6e5eee8311d7bffb2f3bb33301a1ab7de101"), strings.ToLower(submission.Avs))
			assert.Equal(t, strings.ToLower("0x0ddd9dc88e638aef6a8e42d0c98aaa6a48a98d24"), strings.ToLower(submission.Token))
			assert.Equal(t, strings.ToLower("0x7402669fb2c8a0cfe8108acb8a0070257c77ec6906ecb07d97c38e8a5ddc66a9"), strings.ToLower(submission.RewardHash))
			assert.Equal(t, "10000000000000000000000", submission.Amount)
			assert.Equal(t, uint64(2419200), submission.Duration)
			assert.Equal(t, int64(1725494400), submission.StartTimestamp.Unix())
			assert.Equal(t, int64(2419200+1725494400), submission.EndTimestamp.Unix())

			assert.Equal(t, strings.ToLower(strategiesAndMultipliers[i].Strategy), strings.ToLower(submission.Strategy))
			assert.Equal(t, strategiesAndMultipliers[i].Multiplier, submission.Multiplier)
		}

		err = model.CommitFinalState(blockNumber)
		assert.Nil(t, err)

		rewards := make([]*RewardSubmission, 0)
		query := `select * from reward_submissions where block_number = ?`
		res = model.Db.Raw(query, blockNumber).Scan(&rewards)
		assert.Nil(t, res.Error)
		assert.Equal(t, len(strategiesAndMultipliers), len(rewards))

		submissionCounter += len(strategiesAndMultipliers)

		stateRoot, err := model.GenerateStateRoot(blockNumber)
		assert.Nil(t, err)
		assert.NotNil(t, stateRoot)
		assert.True(t, len(stateRoot) > 0)

		teardown(model)
	})

	t.Run("Handle a reward submission for all", func(t *testing.T) {
		blockNumber := uint64(103)

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
			Arguments:        `[{"Name": "submitter", "Type": "address", "Value": "0x66ae7d7c4d492e4e012b95977f14715b74498bc5", "Indexed": true}, {"Name": "submissionNonce", "Type": "uint256", "Value": 3, "Indexed": true}, {"Name": "rewardsSubmissionHash", "Type": "bytes32", "Value": "0x99ebccb0f68eedbf3dff04c7773d6ff94fc439e0eebdd80918b3785ae8099f96", "Indexed": true}, {"Name": "rewardsSubmission", "Type": "((address,uint96)[],address,uint256,uint32,uint32)", "Value": null, "Indexed": false}]`,
			EventName:        "RewardsSubmissionForAllCreated",
			LogIndex:         big.NewInt(12).Uint64(),
			OutputData:       `{"rewardsSubmission": {"token": "0x554c393923c753d146aa34608523ad7946b61662", "amount": 10000000000000000000, "duration": 1814400, "startTimestamp": 1717632000, "strategiesAndMultipliers": [{"strategy": "0xd523267698c81a372191136e477fdebfa33d9fb4", "multiplier": 1000000000000000000}, {"strategy": "0xdccf401fd121d8c542e96bc1d0078884422afad2", "multiplier": 2000000000000000000}]}}`,
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
			{"0xd523267698c81a372191136e477fdebfa33d9fb4", "1000000000000000000"},
			{"0xdccf401fd121d8c542e96bc1d0078884422afad2", "2000000000000000000"},
		}

		typedChange := change.(*RewardSubmissions)
		assert.Equal(t, len(strategiesAndMultipliers), len(typedChange.Submissions))

		for i, submission := range typedChange.Submissions {
			assert.Equal(t, strings.ToLower("0x66ae7d7c4d492e4e012b95977f14715b74498bc5"), strings.ToLower(submission.Avs))
			assert.Equal(t, strings.ToLower("0x554c393923c753d146aa34608523ad7946b61662"), strings.ToLower(submission.Token))
			assert.Equal(t, strings.ToLower("0x99ebccb0f68eedbf3dff04c7773d6ff94fc439e0eebdd80918b3785ae8099f96"), strings.ToLower(submission.RewardHash))
			assert.Equal(t, "10000000000000000000", submission.Amount)
			assert.Equal(t, uint64(1814400), submission.Duration)
			assert.Equal(t, int64(1717632000), submission.StartTimestamp.Unix())
			assert.Equal(t, int64(1814400+1717632000), submission.EndTimestamp.Unix())

			assert.Equal(t, strings.ToLower(strategiesAndMultipliers[i].Strategy), strings.ToLower(submission.Strategy))
			assert.Equal(t, strategiesAndMultipliers[i].Multiplier, submission.Multiplier)
		}

		err = model.CommitFinalState(blockNumber)
		assert.Nil(t, err)

		rewards := make([]*RewardSubmission, 0)
		query := `select * from reward_submissions where block_number = ?`
		res = model.Db.Raw(query, blockNumber).Scan(&rewards)
		assert.Nil(t, res.Error)
		assert.Equal(t, len(strategiesAndMultipliers), len(rewards))

		submissionCounter += len(strategiesAndMultipliers)

		stateRoot, err := model.GenerateStateRoot(blockNumber)
		assert.Nil(t, err)
		assert.NotNil(t, stateRoot)
		assert.True(t, len(stateRoot) > 0)

		teardown(model)
	})
}
