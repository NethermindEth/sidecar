package sqliteContractStore

import (
	"database/sql"
	"errors"
	"fmt"
	"github.com/Layr-Labs/sidecar/internal/contractStore"
	"github.com/Layr-Labs/sidecar/internal/sqlite"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"strings"
)

type SqliteContractStore struct {
	Db     *gorm.DB
	Logger *zap.Logger
}

func NewSqliteContractStore(db *gorm.DB, l *zap.Logger) *SqliteContractStore {
	cs := &SqliteContractStore{
		Db:     db,
		Logger: l,
	}
	return cs
}

func (s *SqliteContractStore) GetContractForAddress(address string) (*contractStore.Contract, error) {
	var contract *contractStore.Contract

	result := s.Db.First(&contract, "contract_address = ?", address)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			s.Logger.Sugar().Debugf("Contract not found in store '%s'", address)
			return nil, nil
		}
		return nil, result.Error
	}

	return contract, nil
}

func (s *SqliteContractStore) FindOrCreateContract(
	address string,
	abiJson string,
	verified bool,
	bytecodeHash string,
	matchingContractAddress string,
) (*contractStore.Contract, bool, error) {
	found := false
	upsertedContract, err := sqlite.WrapTxAndCommit[*contractStore.Contract](func(tx *gorm.DB) (*contractStore.Contract, error) {
		contract := &contractStore.Contract{}
		result := s.Db.First(&contract, "contract_address = ?", strings.ToLower(address))
		if result.Error != nil && !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, result.Error
		}

		// found contract
		if contract.ContractAddress == address {
			found = true
			return contract, nil
		}
		contract = &contractStore.Contract{
			ContractAddress:         strings.ToLower(address),
			ContractAbi:             abiJson,
			Verified:                verified,
			BytecodeHash:            bytecodeHash,
			MatchingContractAddress: matchingContractAddress,
		}

		result = s.Db.Create(contract)
		if result.Error != nil {
			return nil, result.Error
		}

		return contract, nil
	}, nil, s.Db)
	return upsertedContract, found, err
}

func (s *SqliteContractStore) FindVerifiedContractWithMatchingBytecodeHash(bytecodeHash string, address string) (*contractStore.Contract, error) {
	query := `
		select
			*
		from contracts
		where
			bytecode_hash = ?
			and verified = true
			and matching_contract_address = ''
			and contract_address != ?
		order by rowid asc
		limit 1`

	var contract *contractStore.Contract
	result := s.Db.Raw(query, bytecodeHash, address).Scan(&contract)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			s.Logger.Sugar().Debugf("Verified contract not found in store '%s'", bytecodeHash)
			return nil, nil
		}
		return nil, result.Error
	}
	return contract, nil
}

func (s *SqliteContractStore) FindOrCreateProxyContract(
	blockNumber uint64,
	contractAddress string,
	proxyContractAddress string,
) (*contractStore.ProxyContract, bool, error) {
	found := false
	contractAddress = strings.ToLower(contractAddress)
	proxyContractAddress = strings.ToLower(proxyContractAddress)

	upsertedContract, err := sqlite.WrapTxAndCommit[*contractStore.ProxyContract](func(tx *gorm.DB) (*contractStore.ProxyContract, error) {
		contract := &contractStore.ProxyContract{}
		// Proxy contracts are unique on block_number && contract
		result := tx.First(&contract, "contract_address = ? and block_number = ?", contractAddress, blockNumber)
		if result.Error != nil && !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, result.Error
		}

		// found contract
		if contract.ContractAddress == contractAddress {
			found = true
			return contract, nil
		}
		proxyContract := &contractStore.ProxyContract{
			BlockNumber:          int64(blockNumber),
			ContractAddress:      contractAddress,
			ProxyContractAddress: proxyContractAddress,
		}

		result = tx.Model(&contractStore.ProxyContract{}).Clauses(clause.Returning{}).Create(proxyContract)
		if result.Error != nil {
			return nil, result.Error
		}

		return proxyContract, nil
	}, nil, s.Db)
	return upsertedContract, found, err
}

func (s *SqliteContractStore) GetContractWithProxyContract(address string, atBlockNumber uint64) (*contractStore.ContractsTree, error) {
	address = strings.ToLower(address)

	query := `select
		c.contract_address as base_address,
		c.contract_abi as base_abi,
		pcc.contract_address as base_proxy_address,
		pcc.contract_abi as base_proxy_abi,
		pcclike.contract_address as base_proxy_like_address,
		pcclike.contract_abi as base_proxy_like_abi,
		clike.contract_address as base_like_address,
		clike.contract_abi as base_like_abi
	from contracts as c
	left join (
		select
			*
		from proxy_contracts
		where contract_address = @contractAddress and block_number <= @blockNumber
		order by block_number desc limit 1
	) as pc on (1=1)
	left join contracts as pcc on (pcc.contract_address = pc.proxy_contract_address)
	left join contracts as pcclike on (pcc.matching_contract_address = pcclike.contract_address)
	left join contracts as clike on (c.matching_contract_address = clike.contract_address)
	where
		c.contract_address = @contractAddress
	`
	contractTree := &contractStore.ContractsTree{}
	result := s.Db.Raw(query,
		sql.Named("contractAddress", address),
		sql.Named("blockNumber", atBlockNumber),
	).Scan(&contractTree)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			s.Logger.Sugar().Debug(fmt.Sprintf("Contract not found '%s'", address))
			return nil, nil
		}
		return nil, result.Error
	}
	if contractTree.BaseAddress == "" {
		s.Logger.Sugar().Debug(fmt.Sprintf("Contract not found in store '%s'", address))
		return nil, nil
	}

	return contractTree, nil
}

func (s *SqliteContractStore) SetContractCheckedForProxy(address string) (*contractStore.Contract, error) {
	contract := &contractStore.Contract{}

	result := s.Db.Model(contract).
		Clauses(clause.Returning{}).
		Where("contract_address = ?", strings.ToLower(address)).
		Updates(&contractStore.Contract{
			CheckedForProxy: true,
		})

	if result.Error != nil {
		return nil, result.Error
	}

	return contract, nil
}

func (s *SqliteContractStore) SetContractAbi(address string, abi string, verified bool) (*contractStore.Contract, error) {
	contract := &contractStore.Contract{}

	result := s.Db.Model(contract).
		Clauses(clause.Returning{}).
		Where("contract_address = ?", strings.ToLower(address)).
		Updates(&contractStore.Contract{
			ContractAbi:   abi,
			Verified:      verified,
			CheckedForAbi: true,
		})

	if result.Error != nil {
		return nil, result.Error
	}

	return contract, nil
}

func (s *SqliteContractStore) SetContractMatchingContractAddress(address string, matchingContractAddress string) (*contractStore.Contract, error) {
	contract := &contractStore.Contract{}

	result := s.Db.Model(&contract).
		Clauses(clause.Returning{}).
		Where("contract_address = ?", strings.ToLower(address)).
		Updates(&contractStore.Contract{
			MatchingContractAddress: matchingContractAddress,
		})

	if result.Error != nil {
		return nil, result.Error
	}

	return contract, nil
}
