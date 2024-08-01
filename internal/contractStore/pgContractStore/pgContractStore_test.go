package pgContractStore

import (
	"github.com/Layr-Labs/sidecar/internal/config"
	"github.com/Layr-Labs/sidecar/internal/contractStore"
	"github.com/Layr-Labs/sidecar/internal/logger"
	"github.com/Layr-Labs/sidecar/internal/tests"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"testing"
)

func setup() (
	*config.Config,
	*PgContractStore,
	error,
) {
	cfg := tests.GetConfig()
	l, _ := logger.NewLogger(&logger.LoggerConfig{Debug: cfg.Debug})

	_, grm, err := tests.GetDatabaseConnection(cfg)

	pgcs, _ := NewPgContractStore(grm, l)

	return cfg, pgcs, err
}

func TestFindOrCreateContract(t *testing.T) {
	_, pgcs, err := setup()
	if err != nil {
		t.Fatal(err)
	}

	const (
		contractAddress         = "0x2c06ec772df3bbd51a0d1ffa35e07335ce93f414"
		abi                     = "{}"
		verified                = false
		bytecodeHash            = "0x1234"
		matchingContractAddress = ""
	)

	var createdContract *contractStore.Contract
	t.Run("Should create a new address", func(t *testing.T) {
		contract, found, err := pgcs.FindOrCreateContract(contractAddress, abi, verified, bytecodeHash, matchingContractAddress)

		assert.False(t, found)
		assert.Nil(t, err)
		assert.Equal(t, contractAddress, contract.ContractAddress)
		assert.Equal(t, abi, contract.ContractAbi)
		assert.Equal(t, verified, contract.Verified)
		assert.Equal(t, bytecodeHash, contract.BytecodeHash)
		assert.Equal(t, matchingContractAddress, contract.MatchingContractAddress)

		createdContract = contract
	})
	t.Run("Should return existing address", func(t *testing.T) {
		contract, found, err := pgcs.FindOrCreateContract(contractAddress, abi, verified, bytecodeHash, matchingContractAddress)

		assert.True(t, found)
		assert.Nil(t, err)
		assert.Equal(t, contractAddress, contract.ContractAddress)
		assert.Equal(t, contract.Id, createdContract.Id)
		assert.True(t, cmp.Equal(contract, createdContract))
	})
}
