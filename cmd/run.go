package cmd

import (
	"context"
	"fmt"
	"github.com/Layr-Labs/go-sidecar/pkg/clients/ethereum"
	"github.com/Layr-Labs/go-sidecar/pkg/contractCaller"
	"github.com/Layr-Labs/go-sidecar/pkg/contractManager"
	"github.com/Layr-Labs/go-sidecar/pkg/contractStore/postgresContractStore"
	"github.com/Layr-Labs/go-sidecar/pkg/fetcher"
	"github.com/Layr-Labs/go-sidecar/pkg/indexer"
	"github.com/Layr-Labs/go-sidecar/pkg/pipeline"
	"github.com/Layr-Labs/go-sidecar/pkg/postgres"
	"github.com/Layr-Labs/go-sidecar/pkg/shutdown"
	"github.com/Layr-Labs/go-sidecar/pkg/sidecar"
	pgStorage "github.com/Layr-Labs/go-sidecar/pkg/storage/postgres"
	"log"
	"time"

	"github.com/Layr-Labs/go-sidecar/internal/config"
	"github.com/Layr-Labs/go-sidecar/internal/logger"
	"github.com/Layr-Labs/go-sidecar/internal/metrics"
	"github.com/Layr-Labs/go-sidecar/pkg/eigenState/avsOperators"
	"github.com/Layr-Labs/go-sidecar/pkg/eigenState/operatorShares"
	"github.com/Layr-Labs/go-sidecar/pkg/eigenState/rewardSubmissions"
	"github.com/Layr-Labs/go-sidecar/pkg/eigenState/stakerDelegations"
	"github.com/Layr-Labs/go-sidecar/pkg/eigenState/stakerShares"
	"github.com/Layr-Labs/go-sidecar/pkg/eigenState/stateManager"
	"github.com/Layr-Labs/go-sidecar/pkg/eigenState/submittedDistributionRoots"
	"github.com/Layr-Labs/go-sidecar/pkg/postgres/migrations"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run the sidecar",
	Run: func(cmd *cobra.Command, args []string) {
		initRunCmd(cmd)
		cfg := config.NewConfig()
		ctx := context.Background()

		l, _ := logger.NewLogger(&logger.LoggerConfig{Debug: cfg.Debug})

		sdc, err := metrics.InitStatsdClient(cfg.StatsdUrl)
		if err != nil {
			l.Sugar().Fatal("Failed to setup statsd client", zap.Error(err))
		}

		client := ethereum.NewClient(cfg.EthereumRpcConfig.BaseUrl, l)

		pgConfig := postgres.PostgresConfigFromDbConfig(&cfg.DatabaseConfig)
		pgConfig.CreateDbIfNotExists = true

		pg, err := postgres.NewPostgres(pgConfig)
		if err != nil {
			l.Fatal("Failed to setup postgres connection", zap.Error(err))
		}

		grm, err := postgres.NewGormFromPostgresConnection(pg.Db)
		if err != nil {
			l.Fatal("Failed to create gorm instance", zap.Error(err))
		}

		migrator := migrations.NewMigrator(pg.Db, grm, l)
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

		if _, err := avsOperators.NewAvsOperatorsModel(sm, grm, l, cfg); err != nil {
			l.Sugar().Fatalw("Failed to create AvsOperatorsModel", zap.Error(err))
		}
		if _, err := operatorShares.NewOperatorSharesModel(sm, grm, l, cfg); err != nil {
			l.Sugar().Fatalw("Failed to create OperatorSharesModel", zap.Error(err))
		}
		if _, err := stakerDelegations.NewStakerDelegationsModel(sm, grm, l, cfg); err != nil {
			l.Sugar().Fatalw("Failed to create StakerDelegationsModel", zap.Error(err))
		}
		if _, err := stakerShares.NewStakerSharesModel(sm, grm, l, cfg); err != nil {
			l.Sugar().Fatalw("Failed to create StakerSharesModel", zap.Error(err))
		}
		if _, err := submittedDistributionRoots.NewSubmittedDistributionRootsModel(sm, grm, l, cfg); err != nil {
			l.Sugar().Fatalw("Failed to create SubmittedDistributionRootsModel", zap.Error(err))
		}
		if _, err := rewardSubmissions.NewRewardSubmissionsModel(sm, grm, l, cfg); err != nil {
			l.Sugar().Fatalw("Failed to create RewardSubmissionsModel", zap.Error(err))
		}

		fetchr := fetcher.NewFetcher(client, cfg, l)

		cc := contractCaller.NewContractCaller(client, l)

		idxr := indexer.NewIndexer(mds, contractStore, cm, client, fetchr, cc, l, cfg)

		p := pipeline.NewPipeline(fetchr, idxr, mds, sm, l)

		// Create new sidecar instance
		sidecar := sidecar.NewSidecar(&sidecar.SidecarConfig{
			GenesisBlockNumber: cfg.GetGenesisBlockNumber(),
		}, cfg, mds, p, sm, l, client)

		// RPC channel to notify the RPC server to shutdown gracefully
		rpcChannel := make(chan bool)
		err = sidecar.WithRpcServer(ctx, mds, sm, rpcChannel)
		if err != nil {
			l.Sugar().Fatalw("Failed to start RPC server", zap.Error(err))
		}

		// Start the sidecar main process in a goroutine so that we can listen for a shutdown signal
		go sidecar.Start(ctx)

		l.Sugar().Info("Started Sidecar")

		gracefulShutdown := shutdown.CreateGracefulShutdownChannel()

		done := make(chan bool)
		shutdown.ListenForShutdown(gracefulShutdown, done, func() {
			l.Sugar().Info("Shutting down...")
			rpcChannel <- true
			sidecar.ShutdownChan <- true
		}, time.Second*5, l)
	},
}

func initRunCmd(cmd *cobra.Command) {
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		if err := viper.BindPFlag(config.KebabToSnakeCase(f.Name), f); err != nil {
			fmt.Printf("Failed to bind flag '%s' - %+v\n", f.Name, err)
		}
		if err := viper.BindEnv(f.Name); err != nil {
			fmt.Printf("Failed to bind env '%s' - %+v\n", f.Name, err)
		}

	})
}
