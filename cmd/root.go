package cmd

import (
	"github.com/Layr-Labs/sidecar/internal/config"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"os"
	"strings"
)

var rootCmd = &cobra.Command{
	Use:   "sidecar",
	Short: "The EigenLayer Sidecar makes it easy to interact with the EigenLayer protocol data",
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	initConfig(rootCmd)

	rootCmd.PersistentFlags().Bool("debug", false, `"true" or "false"`)
	rootCmd.PersistentFlags().StringP("chain", "c", "mainnet", "The chain to use (mainnet, holesky, preprod")
	rootCmd.PersistentFlags().String("statsd.url", "", `e.g. "localhost:8125"`)

	rootCmd.PersistentFlags().String("ethereum.rpc-url", "", `e.g. "http://34.229.43.36:8545"`)
	rootCmd.PersistentFlags().String("ethereum.ws-url", "", `e.g. "ws://34.229.43.36:8546"`)

	rootCmd.PersistentFlags().String(config.DatabaseHost, "localhost", `Defaults to 'localhost'. Set to something else if you are running PostgreSQL on your own`)
	rootCmd.PersistentFlags().Int(config.DatabasePort, 5432, `Defaults to '5432'`)
	rootCmd.PersistentFlags().String(config.DatabaseUser, "sidecar", `Defaults to 'sidecar'`)
	rootCmd.PersistentFlags().String(config.DatabasePassword, "", ``)
	rootCmd.PersistentFlags().String(config.DatabaseDbName, "sidecar", `Defaults to 'sidecar'`)

	rootCmd.PersistentFlags().Int("rpc.grpc-port", 7100, `e.g. 7100`)
	rootCmd.PersistentFlags().Int("rpc.http-port", 7101, `e.g. 7101`)

	rootCmd.PersistentFlags().VisitAll(func(f *pflag.Flag) {
		key := config.KebabToSnakeCase(f.Name)
		viper.BindPFlag(key, f) //nolint:errcheck
		viper.BindEnv(key)      //nolint:errcheck
	})

	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(runOperatorRestakedStrategiesCmd)
	rootCmd.AddCommand(runVersionCmd)
}

func initConfig(cmd *cobra.Command) {
	viper.SetEnvPrefix(config.ENV_PREFIX)

	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))

	viper.AutomaticEnv()
}
