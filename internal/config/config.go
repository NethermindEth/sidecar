package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type EnvScope string

const ENV_VAR_PREFIX = "SIDE_CAR"

const s3BlobStoreConfigScope EnvScope = "BLOB_STORE_S3"

type Network uint
type Environment int

const (
	Network_Holesky  Network = 0
	Network_Ethereum Network = 1

	Environment_PreProd Environment = 1
	Environment_Testnet Environment = 2
	Environment_Mainnet Environment = 3
)

func ParseNetwork(n string) Network {
	switch n {
	case "mainnet":
		return Network_Ethereum
	default:
		return Network_Holesky
	}
}

func parseEnvironment(env string) Environment {
	switch env {
	case "preprod":
		return Environment_PreProd
	case "testnet":
		return Environment_Testnet
	case "mainnet":
		return Environment_Mainnet
	default:
		return Environment_PreProd
	}
}

func GetEnvironment(e Environment) string {
	switch e {
	case Environment_PreProd:
		return "preprod"
	case Environment_Testnet:
		return "testnet"
	case Environment_Mainnet:
		return "mainnet"
	default:
		return "local"
	}
}

func GetNetwork(n Network) string {
	switch n {
	case Network_Ethereum:
		return "ethereum"
	default:
		return "holesky"
	}
}

func getPrefixedEnvVar(key string) string {
	return os.Getenv(ENV_VAR_PREFIX + "_" + key)
}

func getScopedEnvVar(scope EnvScope, key string) string {
	return getPrefixedEnvVar(fmt.Sprintf("%s_%s", scope, key))
}

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

func parseBooleanEnvVar(envVar string) bool {
	return envVar == "true"
}

func parseIntEnvVar(envVar string, defaultVal int) int {
	if envVar == "" {
		return defaultVal
	}
	val, err := strconv.Atoi(envVar)
	if err != nil {
		return defaultVal
	}
	return val
}

type Config struct {
	Network                    Network
	Environment                Environment
	Debug                      bool
	StatsdUrl                  string
	S3BlobStoreConfig          S3BlobStoreConfig
	EthereumRpcConfig          EthereumRpcConfig
	QuickNodeEthereumRpcConfig EthereumRpcConfig
	PostgresConfig             PostgresConfig
	EtherscanConfig            EtherscanConfig
	RabbitMqConfig             RabbitMqConfig
}

type S3BlobStoreConfig struct {
	Prefix          string
	BucketName      string
	AccessKeyId     string
	SecretAccessKey string
	Region          string
}

type EthereumRpcConfig struct {
	BaseUrl string
	WsUrl   string
}

type PostgresConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	DbName   string
}

type EtherscanConfig struct {
	ApiKeys []string
}

type RabbitMqConfig struct {
	Username      string
	Password      string
	Url           string
	Secure        bool
	PrefetchCount int
}

func NewConfig() *Config {
	return &Config{
		Network:     ParseNetwork(getPrefixedEnvVar("NETWORK")),
		Environment: parseEnvironment(getPrefixedEnvVar("ENVIRONMENT")),
		Debug:       parseBooleanEnvVar(getPrefixedEnvVar("DEBUG")),
		StatsdUrl:   getPrefixedEnvVar("STATSD_URL"),

		S3BlobStoreConfig: S3BlobStoreConfig{
			Prefix:          getScopedEnvVar(s3BlobStoreConfigScope, "PREFIX"),
			BucketName:      getScopedEnvVar(s3BlobStoreConfigScope, "BUCKET_NAME"),
			AccessKeyId:     getScopedEnvVar(s3BlobStoreConfigScope, "ACCESS_KEY_ID"),
			SecretAccessKey: getScopedEnvVar(s3BlobStoreConfigScope, "SECRET_ACCESS_KEY"),
			Region:          getScopedEnvVar(s3BlobStoreConfigScope, "REGION"),
		},

		EthereumRpcConfig: EthereumRpcConfig{
			BaseUrl: getPrefixedEnvVar("ETHEREUM_RPC_BASE_URL"),
			WsUrl:   getPrefixedEnvVar("ETHEREUM_WS_URL"),
		},

		QuickNodeEthereumRpcConfig: EthereumRpcConfig{
			BaseUrl: getPrefixedEnvVar("QUICKNODE_ETHEREUM_RPC_BASE_URL"),
			WsUrl:   getPrefixedEnvVar("QUICKNODE_ETHEREUM_WS_URL"),
		},

		PostgresConfig: PostgresConfig{
			Host:     getPrefixedEnvVar("POSTGRES_HOST"),
			Port:     parseIntEnvVar(getPrefixedEnvVar("POSTGRES_PORT"), 5432),
			Username: getPrefixedEnvVar("POSTGRES_USERNAME"),
			Password: getPrefixedEnvVar("POSTGRES_PASSWORD"),
			DbName:   getPrefixedEnvVar("POSTGRES_DBNAME"),
		},

		EtherscanConfig: EtherscanConfig{
			ApiKeys: parseListEnvVar(getPrefixedEnvVar("ETHERSCAN_API_KEYS")),
		},

		RabbitMqConfig: RabbitMqConfig{
			Username:      getPrefixedEnvVar("RABBITMQ_USERNAME"),
			Password:      getPrefixedEnvVar("RABBITMQ_PASSWORD"),
			Url:           getPrefixedEnvVar("RABBITMQ_URL"),
			Secure:        parseBooleanEnvVar(getPrefixedEnvVar("RABBITMQ_SECURE")),
			PrefetchCount: parseIntEnvVar(getPrefixedEnvVar("RABBITMQ_PREFETCH_COUNT"), 1),
		},
	}
}

func (c *Config) GetAVSDirectoryForEnvAndNetwork() string {
	return AVSDirectoryAddresses[c.Environment][c.Network]
}

var AVSDirectoryAddresses = map[Environment]map[Network]string{
	Environment_PreProd: {
		Network_Holesky: "0x141d6995556135D4997b2ff72EB443Be300353bC",
	},
	Environment_Testnet: {
		Network_Holesky: "0x055733000064333CaDDbC92763c58BF0192fFeBf",
	},
	Environment_Mainnet: {
		Network_Ethereum: "0x135dda560e946695d6f155dacafc6f1f25c1f5af",
	},
}

func (c *Config) GetInterestingAddressForConfigEnv() []string {
	if c.Environment == Environment_PreProd {
		return []string{
			"0xb22ef643e1e067c994019a4c19e403253c05c2b0", // rewards coordinator
			"0xb8d8952f572e67b11e43bc21250967772fa883ff", // eigenpod manager
			"0xf9fbf2e35d8803273e214c99bf15174139f4e67a", // strategy manager
			"0x75dfe5b44c2e530568001400d3f704bc8ae350cc", // delegation manager
			"0x141d6995556135d4997b2ff72eb443be300353bc", // avs directory
		}
	} else if c.Environment == Environment_Testnet {
		return []string{
			"0xacc1fb458a1317e886db376fc8141540537e68fe", // rewards coordinator
			"0x30770d7e3e71112d7a6b7259542d1f680a70e315", // eigenpod manager
			"0xdfb5f6ce42aaa7830e94ecfccad411bef4d4d5b6", // strategy manager
			"0xa44151489861fe9e3055d95adc98fbd462b948e7", // delegation manager
			"0x055733000064333caddbc92763c58bf0192ffebf", // avs directory
		}
	} else if c.Environment == Environment_Mainnet {
		return []string{
			"0x7750d328b314effa365a0402ccfd489b80b0adda", // rewards coordinator
			"0x91e677b07f7af907ec9a428aafa9fc14a0d3a338", // eigenpod manager
			"0x858646372cc42e1a627fce94aa7a7033e7cf075a", // strategy manager
			"0x39053d51b77dc0d36036fc1fcc8cb819df8ef37a", // delegation manager
			"0x135dda560e946695d6f155dacafc6f1f25c1f5af", // avs directory
		}
	} else {
		return []string{}
	}
}

func (c *Config) GetEigenLayerGenesisBlockHeight() (uint64, error) {
	switch c.Environment {
	case Environment_PreProd, Environment_Testnet:
		return 1, nil
	case Environment_Mainnet:
		return 1, nil
	default:
		return 0, fmt.Errorf("unsupported environment %d", c.Environment)
	}
}
