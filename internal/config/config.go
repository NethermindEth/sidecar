package config

import (
	"fmt"
	"github.com/spf13/viper"
	"strings"
)

type EnvScope string

type Network uint
type Environment int

type Chain string

const (
	Chain_Mainnet Chain = "mainnet"
	Chain_Holesky Chain = "holesky"
	Chain_Preprod Chain = "preprod"
)

func parseListEnvVar(envVar string) []string {
	if envVar == "" {
		return []string{}
	}
	// split on commas
	stringList := strings.Split(envVar, ",")

	for i, s := range stringList {
		stringList[i] = strings.TrimSpace(s)
	}
	l := make([]string, 0)
	for _, s := range stringList {
		if s != "" {
			l = append(l, s)
		}
	}
	return l
}

func normalizeFlagName(name string) string {
	return strings.ReplaceAll(name, "-", "_")
}

type Config struct {
	Debug             bool
	StatsdUrl         string
	EthereumRpcConfig EthereumRpcConfig
	EtherscanConfig   EtherscanConfig
	SqliteConfig      SqliteConfig
	RpcConfig         RpcConfig
	Chain             Chain
}

type EthereumRpcConfig struct {
	BaseUrl string
	WsUrl   string
}

type EtherscanConfig struct {
	ApiKeys []string
}

type SqliteConfig struct {
	InMemory   bool
	DbFilePath string
}

type RpcConfig struct {
	GrpcPort int
	HttpPort int
}

func (s *SqliteConfig) GetSqlitePath() string {
	if s.InMemory {
		return "file::memory:?cache=shared"
	}
	return s.DbFilePath
}

func StringWithDefault(value, defaultValue string) string {
	if value == "" {
		return defaultValue
	}
	return value
}

func NewConfig() *Config {
	fmt.Printf("gprc: %+v\n", viper.GetInt64(normalizeFlagName("rpc.grpc-port")))
	return &Config{
		Debug:     viper.GetBool(normalizeFlagName("debug")),
		Chain:     Chain(StringWithDefault(viper.GetString(normalizeFlagName("chain")), "holesky")),
		StatsdUrl: viper.GetString(normalizeFlagName("statsd.url")),

		EthereumRpcConfig: EthereumRpcConfig{
			BaseUrl: viper.GetString(normalizeFlagName("ethereum.rpc_url")),
			WsUrl:   viper.GetString(normalizeFlagName("ethereum.ws_url")),
		},

		EtherscanConfig: EtherscanConfig{
			ApiKeys: parseListEnvVar(viper.GetString(normalizeFlagName("etherscan.api_keys"))),
		},

		SqliteConfig: SqliteConfig{
			InMemory:   viper.GetBool(normalizeFlagName("sqlite.in_memory")),
			DbFilePath: viper.GetString(normalizeFlagName("sqlite.db_file_path")),
		},

		RpcConfig: RpcConfig{
			GrpcPort: viper.GetInt(normalizeFlagName("rpc.grpc_port")),
			HttpPort: viper.GetInt(normalizeFlagName("rpc.http_port")),
		},
	}
}

func (c *Config) GetAVSDirectoryForChain() string {
	return c.GetContractsMapForChain().AvsDirectory
}

var AVSDirectoryAddresses = map[Chain]string{
	Chain_Preprod: "0x141d6995556135D4997b2ff72EB443Be300353bC",
	Chain_Holesky: "0x055733000064333CaDDbC92763c58BF0192fFeBf",
	Chain_Mainnet: "0x135dda560e946695d6f155dacafc6f1f25c1f5af",
}

type ContractAddresses struct {
	RewardsCoordinator string
	EigenpodManager    string
	StrategyManager    string
	DelegationManager  string
	AvsDirectory       string
}

func (c *Config) GetContractsMapForChain() *ContractAddresses {
	if c.Chain == Chain_Preprod {
		return &ContractAddresses{
			RewardsCoordinator: "0xb22ef643e1e067c994019a4c19e403253c05c2b0",
			EigenpodManager:    "0xb8d8952f572e67b11e43bc21250967772fa883ff",
			StrategyManager:    "0xf9fbf2e35d8803273e214c99bf15174139f4e67a",
			DelegationManager:  "0x75dfe5b44c2e530568001400d3f704bc8ae350cc",
			AvsDirectory:       "0x141d6995556135d4997b2ff72eb443be300353bc",
		}
	} else if c.Chain == Chain_Holesky {
		return &ContractAddresses{
			RewardsCoordinator: "0xacc1fb458a1317e886db376fc8141540537e68fe",
			EigenpodManager:    "0x30770d7e3e71112d7a6b7259542d1f680a70e315",
			StrategyManager:    "0xdfb5f6ce42aaa7830e94ecfccad411bef4d4d5b6",
			DelegationManager:  "0xa44151489861fe9e3055d95adc98fbd462b948e7",
			AvsDirectory:       "0x055733000064333caddbc92763c58bf0192ffebf",
		}
	} else if c.Chain == Chain_Mainnet {
		return &ContractAddresses{
			RewardsCoordinator: "0x7750d328b314effa365a0402ccfd489b80b0adda",
			EigenpodManager:    "0x91e677b07f7af907ec9a428aafa9fc14a0d3a338",
			StrategyManager:    "0x858646372cc42e1a627fce94aa7a7033e7cf075a",
			DelegationManager:  "0x39053d51b77dc0d36036fc1fcc8cb819df8ef37a",
			AvsDirectory:       "0x135dda560e946695d6f155dacafc6f1f25c1f5af",
		}
	} else {
		return nil
	}
}

func (c *Config) GetInterestingAddressForConfigEnv() []string {
	addresses := c.GetContractsMapForChain()

	if addresses == nil {
		return []string{}
	}

	return []string{
		addresses.RewardsCoordinator,
		addresses.EigenpodManager,
		addresses.StrategyManager,
		addresses.DelegationManager,
		addresses.AvsDirectory,
	}
}

func (c *Config) GetGenesisBlockNumber() uint64 {
	switch c.Chain {
	case Chain_Preprod:
		return 1140406
	case Chain_Holesky:
		return 1167044
	case Chain_Mainnet:
		return 19492759
	default:
		return 0
	}
}

func (c *Config) GetEigenLayerGenesisBlockHeight() (uint64, error) {
	switch c.Chain {
	case Chain_Preprod, Chain_Holesky:
		return 1, nil
	case Chain_Mainnet:
		return 1, nil
	default:
		return 0, fmt.Errorf("unsupported chain %s", c.Chain)
	}
}

func (c *Config) GetOperatorRestakedStrategiesStartBlock() int64 {
	switch c.Chain {
	case Chain_Preprod:
	case Chain_Holesky:
		return 1162800
	case Chain_Mainnet:
		return 19616400
	}
	return 0
}

func KebabToSnakeCase(str string) string {
	return strings.ReplaceAll(str, "-", "_")
}
