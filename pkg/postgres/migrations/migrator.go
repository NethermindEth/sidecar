package migrations

import (
	"database/sql"
	"fmt"
	_202409061249_bootstrapDb "github.com/Layr-Labs/go-sidecar/pkg/postgres/migrations/202409061249_bootstrapDb"
	_202409061250_eigenlayerStateTables "github.com/Layr-Labs/go-sidecar/pkg/postgres/migrations/202409061250_eigenlayerStateTables"
	_202409061720_operatorShareChanges "github.com/Layr-Labs/go-sidecar/pkg/postgres/migrations/202409061720_operatorShareChanges"
	_202409062151_stakerDelegations "github.com/Layr-Labs/go-sidecar/pkg/postgres/migrations/202409062151_stakerDelegations"
	_202409080918_staterootTable "github.com/Layr-Labs/go-sidecar/pkg/postgres/migrations/202409080918_staterootTable"
	_202409082234_stakerShare "github.com/Layr-Labs/go-sidecar/pkg/postgres/migrations/202409082234_stakerShare"
	_202409101144_submittedDistributionRoot "github.com/Layr-Labs/go-sidecar/pkg/postgres/migrations/202409101144_submittedDistributionRoot"
	_202409101540_rewardSubmissions "github.com/Layr-Labs/go-sidecar/pkg/postgres/migrations/202409101540_rewardSubmissions"
	_202409161057_avsOperatorDeltas "github.com/Layr-Labs/go-sidecar/pkg/postgres/migrations/202409161057_avsOperatorDeltas"
	_202409181340_stakerDelegationDelta "github.com/Layr-Labs/go-sidecar/pkg/postgres/migrations/202409181340_stakerDelegationDelta"
	_202410241239_combinedRewards "github.com/Layr-Labs/go-sidecar/pkg/postgres/migrations/202410241239_combinedRewards"
	_202410241313_operatorAvsRegistrationSnapshots "github.com/Layr-Labs/go-sidecar/pkg/postgres/migrations/202410241313_operatorAvsRegistrationSnapshots"
	_202410241417_operatorAvsStrategySnapshots "github.com/Layr-Labs/go-sidecar/pkg/postgres/migrations/202410241417_operatorAvsStrategySnapshots"
	_202410241431_operatorShareSnapshots "github.com/Layr-Labs/go-sidecar/pkg/postgres/migrations/202410241431_operatorShareSnapshots"
	_202410241450_stakerDelegationSnapshots "github.com/Layr-Labs/go-sidecar/pkg/postgres/migrations/202410241450_stakerDelegationSnapshots"
	_202410241456_stakerShareSnapshots "github.com/Layr-Labs/go-sidecar/pkg/postgres/migrations/202410241456_stakerShareSnapshots"
	_202410241539_goldTables "github.com/Layr-Labs/go-sidecar/pkg/postgres/migrations/202410241539_goldTables"
	_202410301449_generatedRewardsSnapshots "github.com/Layr-Labs/go-sidecar/pkg/postgres/migrations/202410301449_generatedRewardsSnapshots"
	_202411041043_blockNumberFkConstraint "github.com/Layr-Labs/go-sidecar/pkg/postgres/migrations/202411041043_blockNumberFkConstraint"
	_202411041332_stakerShareDeltaBlockFk "github.com/Layr-Labs/go-sidecar/pkg/postgres/migrations/202411041332_stakerShareDeltaBlockFk"
	_202411042033_cleanupDuplicates "github.com/Layr-Labs/go-sidecar/pkg/postgres/migrations/202411042033_cleanupDuplicates"
	_202411051308_submittedDistributionRootIndex "github.com/Layr-Labs/go-sidecar/pkg/postgres/migrations/202411051308_submittedDistributionRootIndex"
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
	_ = gDb.AutoMigrate(&Migrations{})
	return &Migrator{
		Db:     db,
		GDb:    gDb,
		Logger: l,
	}
}

func (m *Migrator) MigrateAll() error {
	migrations := []Migration{
		&_202409061249_bootstrapDb.Migration{},
		&_202409061250_eigenlayerStateTables.Migration{},
		&_202409061720_operatorShareChanges.Migration{},
		&_202409062151_stakerDelegations.Migration{},
		&_202409080918_staterootTable.Migration{},
		&_202409082234_stakerShare.Migration{},
		&_202409101144_submittedDistributionRoot.Migration{},
		&_202409101540_rewardSubmissions.Migration{},
		&_202409161057_avsOperatorDeltas.Migration{},
		&_202409181340_stakerDelegationDelta.Migration{},
		&_202410241239_combinedRewards.Migration{},
		&_202410241313_operatorAvsRegistrationSnapshots.Migration{},
		&_202410241417_operatorAvsStrategySnapshots.Migration{},
		&_202410241431_operatorShareSnapshots.Migration{},
		&_202410241450_stakerDelegationSnapshots.Migration{},
		&_202410241456_stakerShareSnapshots.Migration{},
		&_202410241539_goldTables.Migration{},
		&_202410301449_generatedRewardsSnapshots.Migration{},
		&_202411041043_blockNumberFkConstraint.Migration{},
		&_202411041332_stakerShareDeltaBlockFk.Migration{},
		&_202411042033_cleanupDuplicates.Migration{},
		&_202411051308_submittedDistributionRootIndex.Migration{},
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
