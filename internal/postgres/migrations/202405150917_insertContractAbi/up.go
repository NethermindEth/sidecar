package _202405150917_insertContractAbi

import (
	"database/sql"
	"github.com/Layr-Labs/sidecar/internal/contractStore"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"os"
	"strings"
)

type Migration struct {
}

func (m *Migration) Up(db *sql.DB, grm *gorm.DB) error {
	pcJson, err := os.ReadFile("./abi/paymentCoordinator.abi")
	if err != nil {
		return err
	}

	contractAddress := strings.ToLower("0xd73A2490E286e50efD167a39302a2701BA40231E")

	contract := &contractStore.Contract{
		ContractAddress: contractAddress,
		ContractAbi:     string(pcJson),
	}

	result := grm.Model(&contractStore.Contract{}).Clauses(clause.Returning{}).Create(contract)

	if result.Error != nil {
		return result.Error
	}
	return nil
}

func (m *Migration) GetName() string {
	return "202405150917_insertContractAbi"
}
