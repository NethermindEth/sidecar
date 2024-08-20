package migrations

import (
	"database/sql"
	"fmt"
	_202405150900_bootstrapDb "github.com/Layr-Labs/sidecar/internal/postgres/migrations/202405150900_bootstrapDb"
	_202405150917_insertContractAbi "github.com/Layr-Labs/sidecar/internal/postgres/migrations/202405150917_insertContractAbi"
	_202405151523_addTransactionToFrom "github.com/Layr-Labs/sidecar/internal/postgres/migrations/202405151523_addTransactionToFrom"
	_202405170842_addBlockInfoToTransactionLog "github.com/Layr-Labs/sidecar/internal/postgres/migrations/202405170842_addBlockInfoToTransactionLog"
	_202405171056_unverifiedContracts "github.com/Layr-Labs/sidecar/internal/postgres/migrations/202405171056_unverifiedContracts"
	_202405171345_addUpdatedPaymentCoordinatorAbi "github.com/Layr-Labs/sidecar/internal/postgres/migrations/202405171345_addUpdatedPaymentCoordinatorAbi"
	_202405201503_fixTransactionHashConstraint "github.com/Layr-Labs/sidecar/internal/postgres/migrations/202405201503_fixTransactionHashConstraint"
	_202405300925_addUniqueBlockConstraint "github.com/Layr-Labs/sidecar/internal/postgres/migrations/202405300925_addUniqueBlockConstraint"
	_202405312008_indexTransactionContractAddress "github.com/Layr-Labs/sidecar/internal/postgres/migrations/202405312008_indexTransactionContractAddress"
	_202405312134_handleProxyContracts "github.com/Layr-Labs/sidecar/internal/postgres/migrations/202405312134_handleProxyContracts"
	_202406030920_addCheckedForProxyFlag "github.com/Layr-Labs/sidecar/internal/postgres/migrations/202406030920_addCheckedForProxyFlag"
	_202406031946_addSerialIdToContracts "github.com/Layr-Labs/sidecar/internal/postgres/migrations/202406031946_addSerialIdToContracts"
	_202406051937_addBytecodeIndex "github.com/Layr-Labs/sidecar/internal/postgres/migrations/202406051937_addBytecodeIndex"
	_202406071318_indexTransactionLogBlockNumber "github.com/Layr-Labs/sidecar/internal/postgres/migrations/202406071318_indexTransactionLogBlockNumber"
	_202406110848_transactionLogsContractIndex "github.com/Layr-Labs/sidecar/internal/postgres/migrations/202406110848_transactionLogsContractIndex"
	_202406141007_addCheckedForAbiFlag "github.com/Layr-Labs/sidecar/internal/postgres/migrations/202406141007_addCheckedForAbiFlag"
	_202406251424_addTransactionLogsOutputDataColumn "github.com/Layr-Labs/sidecar/internal/postgres/migrations/202406251424_addTransactionLogsOutputDataColumn"
	_202406251426_addTransactionIndexes "github.com/Layr-Labs/sidecar/internal/postgres/migrations/202406251426_addTransactionIndexes"
	_202407101440_addOperatorRestakedStrategiesTable "github.com/Layr-Labs/sidecar/internal/postgres/migrations/202407101440_addOperatorRestakedStrategiesTable"
	_202407110946_addBlockTimeToRestakedStrategies "github.com/Layr-Labs/sidecar/internal/postgres/migrations/202407110946_addBlockTimeToRestakedStrategies"
	_202407111116_addAvsDirectoryAddress "github.com/Layr-Labs/sidecar/internal/postgres/migrations/202407111116_addAvsDirectoryAddress"
	_202407121407_updateProxyContractIndex "github.com/Layr-Labs/sidecar/internal/postgres/migrations/202407121407_updateProxyContractIndex"
	_202408200934_eigenlayerStateTables "github.com/Layr-Labs/sidecar/internal/postgres/migrations/202408200934_eigenlayerStateTables"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"time"
)

type Migration interface {
	Up(db *sql.DB, grm *gorm.DB) error
	GetName() string
}

type Migrator struct {
	Db     *sql.DB
	GDb    *gorm.DB
	Logger *zap.Logger
}

func NewMigrator(db *sql.DB, gDb *gorm.DB, l *zap.Logger) *Migrator {
	gDb.AutoMigrate(&Migrations{})
	return &Migrator{
		Db:     db,
		GDb:    gDb,
		Logger: l,
	}
}

func (m *Migrator) MigrateAll() error {
	migrations := []Migration{
		&_202405150900_bootstrapDb.Migration{},
		&_202405150917_insertContractAbi.Migration{},
		&_202405151523_addTransactionToFrom.Migration{},
		&_202405170842_addBlockInfoToTransactionLog.Migration{},
		&_202405171056_unverifiedContracts.Migration{},
		&_202405171345_addUpdatedPaymentCoordinatorAbi.Migration{},
		&_202405201503_fixTransactionHashConstraint.Migration{},
		&_202405300925_addUniqueBlockConstraint.Migration{},
		&_202405312008_indexTransactionContractAddress.Migration{},
		&_202405312134_handleProxyContracts.Migration{},
		&_202406030920_addCheckedForProxyFlag.Migration{},
		&_202406031946_addSerialIdToContracts.Migration{},
		&_202406051937_addBytecodeIndex.Migration{},
		&_202406071318_indexTransactionLogBlockNumber.Migration{},
		&_202406110848_transactionLogsContractIndex.Migration{},
		&_202406141007_addCheckedForAbiFlag.Migration{},
		&_202406251424_addTransactionLogsOutputDataColumn.Migration{},
		&_202406251426_addTransactionIndexes.Migration{},
		&_202407101440_addOperatorRestakedStrategiesTable.Migration{},
		&_202407110946_addBlockTimeToRestakedStrategies.Migration{},
		&_202407111116_addAvsDirectoryAddress.Migration{},
		&_202407121407_updateProxyContractIndex.Migration{},
		&_202408200934_eigenlayerStateTables.Migration{},
	}

	for _, migration := range migrations {
		err := m.Migrate(migration)
		if err != nil {
			panic(err)
		}
	}
	return nil
}

func (m *Migrator) Migrate(migration Migration) error {
	name := migration.GetName()

	// find migration by name
	var migrationRecord Migrations
	result := m.GDb.Find(&migrationRecord, "name = ?", name).Limit(1)

	if result.Error == nil && result.RowsAffected == 0 {
		m.Logger.Sugar().Infof("Running migration '%s'", name)
		// run migration
		err := migration.Up(m.Db, m.GDb)
		if err != nil {
			m.Logger.Sugar().Errorw(fmt.Sprintf("Failed to run migration '%s'", name), zap.Error(err))
			return err
		}

		// record migration
		migrationRecord = Migrations{
			Name: name,
		}
		result = m.GDb.Create(&migrationRecord)
		if result.Error != nil {
			m.Logger.Sugar().Errorw(fmt.Sprintf("Failed to record migration '%s'", name), zap.Error(result.Error))
			return result.Error
		}
	} else if result.Error != nil {
		m.Logger.Sugar().Errorw(fmt.Sprintf("Failed to find migration '%s'", name), zap.Error(result.Error))
		return result.Error
	} else if result.RowsAffected > 0 {
		m.Logger.Sugar().Infof("Migration %s already run", name)
		return nil
	}
	return nil
}

type Migrations struct {
	Name      string    `gorm:"primaryKey"`
	CreatedAt time.Time `gorm:"default:current_timestamp;type:timestamp with time zone"`
	UpdatedAt time.Time `gorm:"default:null;type:timestamp with time zone"`
}
