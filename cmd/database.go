package cmd

import (
	"fmt"
	"github.com/Layr-Labs/sidecar/internal/config"
	"github.com/Layr-Labs/sidecar/internal/logger"
	"github.com/Layr-Labs/sidecar/internal/metrics"
	"github.com/Layr-Labs/sidecar/internal/version"
	"github.com/Layr-Labs/sidecar/pkg/clients/ethereum"
	"github.com/Layr-Labs/sidecar/pkg/contractCaller/sequentialContractCaller"
	"github.com/Layr-Labs/sidecar/pkg/contractManager"
	"github.com/Layr-Labs/sidecar/pkg/contractStore/postgresContractStore"
	"github.com/Layr-Labs/sidecar/pkg/eigenState"
	"github.com/Layr-Labs/sidecar/pkg/eigenState/stateManager"
	"github.com/Layr-Labs/sidecar/pkg/fetcher"
	"github.com/Layr-Labs/sidecar/pkg/indexer"
	"github.com/Layr-Labs/sidecar/pkg/pipeline"
	"github.com/Layr-Labs/sidecar/pkg/postgres"
	"github.com/Layr-Labs/sidecar/pkg/postgres/migrations"
	"github.com/Layr-Labs/sidecar/pkg/rewards"
	"github.com/Layr-Labs/sidecar/pkg/rewards/stakerOperators"
	pgStorage "github.com/Layr-Labs/sidecar/pkg/storage/postgres"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"log"
)

var runDatabaseCmd = &cobra.Command{
	Use:   "database",
	Short: "Database utility",
	Run: func(cmd *cobra.Command, args []string) {
		initDatabaseCmd(cmd)
		cfg := config.NewConfig()

		l, _ := logger.NewLogger(&logger.LoggerConfig{Debug: cfg.Debug})

		l.Sugar().Infow("sidecar run",
			zap.String("version", version.GetVersion()),
			zap.String("commit", version.GetCommit()),
		)

		metricsClients, err := metrics.InitMetricsSinksFromConfig(cfg, l)
		if err != nil {
			l.Sugar().Fatal("Failed to setup metrics sink", zap.Error(err))
		}

		sdc, err := metrics.NewMetricsSink(&metrics.MetricsSinkConfig{}, metricsClients)
		if err != nil {
			l.Sugar().Fatal("Failed to setup metrics sink", zap.Error(err))
		}

		client := ethereum.NewClient(ethereum.ConvertGlobalConfigToEthereumConfig(&cfg.EthereumRpcConfig), l)

		pgConfig := postgres.PostgresConfigFromDbConfig(&cfg.DatabaseConfig)

		pg, err := postgres.NewPostgres(pgConfig)
		if err != nil {
			l.Fatal("Failed to setup postgres connection", zap.Error(err))
		}

		grm, err := postgres.NewGormFromPostgresConnection(pg.Db)
		if err != nil {
			l.Fatal("Failed to create gorm instance", zap.Error(err))
		}

		migrator := migrations.NewMigrator(pg.Db, grm, l, cfg)
		if err = migrator.MigrateAll(); err != nil {
			l.Fatal("Failed to migrate", zap.Error(err))
		}

		contractStore := postgresContractStore.NewPostgresContractStore(grm, l, cfg)
		if err := contractStore.InitializeCoreContracts(); err != nil {
			log.Fatalf("Failed to initialize core contracts: %v", err)
		}

		cm := contractManager.NewContractManager(contractStore, client, sdc, l)

		mds := pgStorage.NewPostgresBlockStore(grm, l, cfg)
		if err != nil {
			log.Fatalln(err)
		}

		sm := stateManager.NewEigenStateManager(l, grm)

		if err := eigenState.LoadEigenStateModels(sm, grm, l, cfg); err != nil {
			l.Sugar().Fatalw("Failed to load eigen state models", zap.Error(err))
		}

		fetchr := fetcher.NewFetcher(client, cfg, l)

		cc := sequentialContractCaller.NewSequentialContractCaller(client, cfg, cfg.EthereumRpcConfig.ContractCallBatchSize, l)

		idxr := indexer.NewIndexer(mds, contractStore, cm, client, fetchr, cc, grm, l, cfg)

		sog := stakerOperators.NewStakerOperatorGenerator(grm, l, cfg)

		rc, err := rewards.NewRewardsCalculator(cfg, grm, mds, sog, l)
		if err != nil {
			l.Sugar().Fatalw("Failed to create rewards calculator", zap.Error(err))
		}

		_ = pipeline.NewPipeline(fetchr, idxr, mds, sm, rc, cfg, sdc, l)

		l.Sugar().Infow("Done")
	},
}

func initDatabaseCmd(cmd *cobra.Command) {
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		if err := viper.BindPFlag(config.KebabToSnakeCase(f.Name), f); err != nil {
			fmt.Printf("Failed to bind flag '%s' - %+v\n", f.Name, err)
		}
		if err := viper.BindEnv(f.Name); err != nil {
			fmt.Printf("Failed to bind env '%s' - %+v\n", f.Name, err)
		}

	})
}
