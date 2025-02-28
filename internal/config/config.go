package config

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/viper"
)

type EnvScope string

type Network uint
type Environment int

type Chain string

func (c Chain) String() string {
	return string(c)
}

type ForkName string

const (
	Chain_Mainnet Chain = "mainnet"
	Chain_Holesky Chain = "holesky"
	Chain_Preprod Chain = "preprod"

	ENV_PREFIX = "SIDECAR"
)

// Rewards forks named after rivers
const (
	RewardsFork_Nile    ForkName = "nile"
	RewardsFork_Amazon  ForkName = "amazon"
	RewardsFork_Panama  ForkName = "panama"
	RewardsFork_Arno    ForkName = "arno"
	RewardsFork_Trinity ForkName = "trinity"
)

func normalizeFlagName(name string) string {
	return strings.ReplaceAll(name, "-", "_")
}

type EthereumRpcConfig struct {
	BaseUrl               string
	ContractCallBatchSize int  // Number of contract calls to make in parallel
	UseNativeBatchCall    bool // Use the native eth_call method for batch calls
	NativeBatchCallSize   int  // Number of calls to put in a single eth_call request
	ChunkedBatchCallSize  int  // Number of calls to make in parallel
}

type DatabaseConfig struct {
	Host        string
	Port        int
	User        string
	Password    string
	DbName      string
	SchemaName  string
	SSLMode     string // disable, require, verify-ca, verify-full
	SSLCert     string // path to client certificate
	SSLKey      string // path to client private key
	SSLRootCert string // path to root certificate
}

type SnapshotConfig struct {
	OutputFile string
	InputFile  string
}

type CreateSnapshotConfig struct {
	OutputFile           string
	GenerateMetadataFile bool
	Kind                 string
}

type RestoreSnapshotConfig struct {
	InputFile       string
	VerifyHash      bool
	VerifySignature bool
	ManifestUrl     string
	Kind            string
}

type RpcConfig struct {
	GrpcPort int
	HttpPort int
}

type RewardsConfig struct {
	ValidateRewardsRoot          bool
	GenerateStakerOperatorsTable bool
}

type StatsdConfig struct {
	Enabled    bool
	Url        string
	SampleRate float64
}

type DataDogConfig struct {
	StatsdConfig StatsdConfig
}

type PrometheusConfig struct {
	Enabled bool
	Port    int
}

type SidecarPrimaryConfig struct {
	Url       string
	Secure    bool
	IsPrimary bool
}

type IpfsConfig struct {
	Url string
}

type EtherscanConfig struct {
	ApiKey string
}

type Config struct {
	Debug                 bool
	EthereumRpcConfig     EthereumRpcConfig
	DatabaseConfig        DatabaseConfig
	CreateSnapshotConfig  CreateSnapshotConfig
	RestoreSnapshotConfig RestoreSnapshotConfig
	RpcConfig             RpcConfig
	Chain                 Chain
	Rewards               RewardsConfig
	DataDogConfig         DataDogConfig
	PrometheusConfig      PrometheusConfig
	SidecarPrimaryConfig  SidecarPrimaryConfig
	IpfsConfig            IpfsConfig
	EtherscanConfig       EtherscanConfig
}

func StringWithDefault(value, defaultValue string) string {
	if value == "" {
		return defaultValue
	}
	return value
}

func StringWithDefaults(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

var (
	Debug               = "debug"
	DatabaseHost        = "database.host"
	DatabasePort        = "database.port"
	DatabaseUser        = "database.user"
	DatabasePassword    = "database.password"
	DatabaseDbName      = "database.db_name"
	DatabaseSchemaName  = "database.schema_name"
	DatabaseSSLMode     = "database.ssl_mode"
	DatabaseSSLCert     = "database.ssl_cert"
	DatabaseSSLKey      = "database.ssl_key"
	DatabaseSSLRootCert = "database.ssl_root_cert"

	SnapshotOutputFile = "output_file"
	SnapshotOutput     = "output"
	SnapshotKind       = "kind"

	SnapshotInputFile       = "input_file"
	SnapshotInput           = "input"
	SnapshotVerifyHash      = "verify-hash"
	SnapshotVerifySignature = "verify-signature"
	SnapshotManifestUrl     = "manifest-url"

	SnapshotOutputMetadataFile = "generate-metadata-file"

	RewardsValidateRewardsRoot          = "rewards.validate_rewards_root"
	RewardsGenerateStakerOperatorsTable = "rewards.generate_staker_operators_table"

	EthereumRpcBaseUrl               = "ethereum.rpc_url"
	EthereumRpcContractCallBatchSize = "ethereum.contract_call_batch_size"
	EthereumRpcUseNativeBatchCall    = "ethereum.use_native_batch_call"
	EthereumRpcNativeBatchCallSize   = "ethereum.native_batch_call_size"
	EthereumRpcChunkedBatchCallSize  = "ethereum.chunked_batch_call_size"

	DataDogStatsdEnabled    = "datadog.statsd.enabled"
	DataDogStatsdUrl        = "datadog.statsd.url"
	DataDogStatsdSampleRate = "datadog.statsd.sample-rate"

	PrometheusEnabled = "prometheus.enabled"
	PrometheusPort    = "prometheus.port"

	SidecarPrimaryUrl = "sidecar-primary.url"

	IpfsUrl = "ipfs.url"

	EtherscanApiKey = "etherscan.api-key"
)

func NewConfig() *Config {
	return &Config{
		Debug: viper.GetBool(normalizeFlagName("debug")),
		Chain: Chain(StringWithDefault(viper.GetString(normalizeFlagName("chain")), "holesky")),

		EthereumRpcConfig: EthereumRpcConfig{
			BaseUrl:               viper.GetString(normalizeFlagName(EthereumRpcBaseUrl)),
			ContractCallBatchSize: viper.GetInt(normalizeFlagName(EthereumRpcContractCallBatchSize)),
			UseNativeBatchCall:    viper.GetBool(normalizeFlagName(EthereumRpcUseNativeBatchCall)),
			NativeBatchCallSize:   viper.GetInt(normalizeFlagName(EthereumRpcNativeBatchCallSize)),
			ChunkedBatchCallSize:  viper.GetInt(normalizeFlagName(EthereumRpcChunkedBatchCallSize)),
		},

		DatabaseConfig: DatabaseConfig{
			Host:        viper.GetString(normalizeFlagName(DatabaseHost)),
			Port:        viper.GetInt(normalizeFlagName(DatabasePort)),
			User:        viper.GetString(normalizeFlagName(DatabaseUser)),
			Password:    viper.GetString(normalizeFlagName(DatabasePassword)),
			DbName:      viper.GetString(normalizeFlagName(DatabaseDbName)),
			SchemaName:  viper.GetString(normalizeFlagName(DatabaseSchemaName)),
			SSLMode:     StringWithDefault(viper.GetString(normalizeFlagName(DatabaseSSLMode)), "disable"),
			SSLCert:     viper.GetString(normalizeFlagName(DatabaseSSLCert)),
			SSLKey:      viper.GetString(normalizeFlagName(DatabaseSSLKey)),
			SSLRootCert: viper.GetString(normalizeFlagName(DatabaseSSLRootCert)),
		},

		CreateSnapshotConfig: CreateSnapshotConfig{
			OutputFile:           StringWithDefaults(viper.GetString(normalizeFlagName(SnapshotOutput)), viper.GetString(normalizeFlagName(SnapshotOutputFile))),
			GenerateMetadataFile: viper.GetBool(normalizeFlagName(SnapshotOutputMetadataFile)),
			Kind:                 StringWithDefault(viper.GetString(normalizeFlagName(SnapshotKind)), "full"),
		},

		RestoreSnapshotConfig: RestoreSnapshotConfig{
			InputFile:       StringWithDefaults(viper.GetString(normalizeFlagName(SnapshotInput)), viper.GetString(normalizeFlagName(SnapshotInputFile))),
			VerifyHash:      viper.GetBool(normalizeFlagName(SnapshotVerifyHash)),
			VerifySignature: viper.GetBool(normalizeFlagName(SnapshotVerifySignature)),
			ManifestUrl:     viper.GetString(normalizeFlagName(SnapshotManifestUrl)),
			Kind:            StringWithDefault(viper.GetString(normalizeFlagName(SnapshotKind)), "full"),
		},

		RpcConfig: RpcConfig{
			GrpcPort: viper.GetInt(normalizeFlagName("rpc.grpc_port")),
			HttpPort: viper.GetInt(normalizeFlagName("rpc.http_port")),
		},

		Rewards: RewardsConfig{
			ValidateRewardsRoot:          viper.GetBool(normalizeFlagName(RewardsValidateRewardsRoot)),
			GenerateStakerOperatorsTable: viper.GetBool(normalizeFlagName(RewardsGenerateStakerOperatorsTable)),
		},

		DataDogConfig: DataDogConfig{
			StatsdConfig: StatsdConfig{
				Enabled:    viper.GetBool(normalizeFlagName(DataDogStatsdEnabled)),
				Url:        viper.GetString(normalizeFlagName(DataDogStatsdUrl)),
				SampleRate: viper.GetFloat64(normalizeFlagName(DataDogStatsdSampleRate)),
			},
		},

		PrometheusConfig: PrometheusConfig{
			Enabled: viper.GetBool(normalizeFlagName(PrometheusEnabled)),
			Port:    viper.GetInt(normalizeFlagName(PrometheusPort)),
		},

		SidecarPrimaryConfig: SidecarPrimaryConfig{
			Url: viper.GetString(normalizeFlagName(SidecarPrimaryUrl)),
		},

		IpfsConfig: IpfsConfig{
			Url: StringWithDefault(viper.GetString(normalizeFlagName(IpfsUrl)), "https://ipfs.io/ipfs"),
		},

		EtherscanConfig: EtherscanConfig{
			ApiKey: viper.GetString(normalizeFlagName(EtherscanApiKey)),
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

func (c *Config) ChainIsOneOf(chains ...Chain) bool {
	return slices.Contains(chains, c.Chain)
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
		return 17445563
	default:
		return 0
	}
}

type ForkMap map[ForkName]string

func (c *Config) GetRewardsSqlForkDates() (ForkMap, error) {
	switch c.Chain {
	case Chain_Preprod:
		return ForkMap{
			RewardsFork_Amazon:  "1970-01-01", // Amazon hard fork was never on preprod as we backfilled
			RewardsFork_Nile:    "2024-08-14", // Last calculation end timestamp was 8-13: https://holesky.etherscan.io/tx/0xb5a6855e88c79312b7c0e1c9f59ae9890b97f157ea27e69e4f0fadada4712b64#eventlog
			RewardsFork_Panama:  "2024-10-01",
			RewardsFork_Arno:    "2024-12-11",
			RewardsFork_Trinity: "2025-01-09",
		}, nil
	case Chain_Holesky:
		return ForkMap{
			RewardsFork_Amazon:  "1970-01-01", // Amazon hard fork was never on testnet as we backfilled
			RewardsFork_Nile:    "2024-08-13", // Last calculation end timestamp was 8-12: https://holesky.etherscan.io/tx/0x5fc81b5ed2a78b017ef313c181d8627737a97fef87eee85acedbe39fc8708c56#eventlog
			RewardsFork_Panama:  "2024-10-01",
			RewardsFork_Arno:    "2024-12-13",
			RewardsFork_Trinity: "2025-01-09",
		}, nil
	case Chain_Mainnet:
		return ForkMap{
			RewardsFork_Amazon:  "2024-08-02", // Last calculation end timestamp was 8-01: https://etherscan.io/tx/0x2aff6f7b0132092c05c8f6f41a5e5eeeb208aa0d95ebcc9022d7823e343dd012#eventlog
			RewardsFork_Nile:    "2024-08-12", // Last calculation end timestamp was 8-11: https://etherscan.io/tx/0x922d29d93c02d189fc2332041f01a80e0007cd7a625a5663ef9d30082f7ef66f#eventlog
			RewardsFork_Panama:  "2024-10-01",
			RewardsFork_Arno:    "2025-01-21",
			RewardsFork_Trinity: "2025-01-21",
		}, nil
	}
	return nil, errors.New("unsupported chain")
}

type ModelForkMap map[ForkName]uint64

// Model forks, named after US capitols
const (
	// ModelFork_Austin changes the formatting for merkel leaves in: ODRewardSubmissions, OperatorAVSSplits, and
	// OperatorPISplits based on feedback from the rewards-v2 audit
	ModelFork_Austin ForkName = "austin"
)

func (c *Config) GetModelForks() (ModelForkMap, error) {
	switch c.Chain {
	case Chain_Preprod:
		return ModelForkMap{
			ModelFork_Austin: 3113600,
		}, nil
	case Chain_Holesky:
		return ModelForkMap{
			ModelFork_Austin: 3113600,
		}, nil
	case Chain_Mainnet:
		return ModelForkMap{
			ModelFork_Austin: 0, // doesnt apply to mainnet
		}, nil
	}
	return nil, errors.New("unsupported chain")

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

func (c *Config) GetOperatorRestakedStrategiesStartBlock() uint64 {
	switch c.Chain {
	case Chain_Preprod:
	case Chain_Holesky:
		return 1162800
	case Chain_Mainnet:
		return 19616400
	}
	return 0
}

func (c *Config) IsRewardsV2EnabledForCutoffDate(cutoffDate string) (bool, error) {
	forks, err := c.GetRewardsSqlForkDates()
	if err != nil {
		return false, err
	}
	cutoffDateTime, err := time.Parse(time.DateOnly, cutoffDate)
	if err != nil {
		return false, errors.Join(fmt.Errorf("failed to parse cutoff date %s", cutoffDate), err)
	}
	arnoForkDateTime, err := time.Parse(time.DateOnly, forks[RewardsFork_Arno])
	if err != nil {
		return false, errors.Join(fmt.Errorf("failed to parse Arno fork date %s", forks[RewardsFork_Arno]), err)
	}

	return cutoffDateTime.Compare(arnoForkDateTime) >= 0, nil
}

// CanIgnoreIncorrectRewardsRoot returns true if the rewards root can be ignored for the given block number
//
// Due to inconsistencies in the rewards root calculation on testnet, we know that some roots
// are not fully reproducible and can be ignored.
func (c *Config) CanIgnoreIncorrectRewardsRoot(blockNumber uint64) bool {
	switch c.Chain {
	case Chain_Preprod:
		// roughly 2024-08-01
		if blockNumber < 2046020 {
			return true
		}
		// test root posted that was invalid for 2024-11-23 (cutoff date 2024-11-24)
		if blockNumber == 2812052 {
			return true
		}

		// ignore rewards-v2 and slashing deployment/testing range
		if blockNumber >= 2877938 && blockNumber <= 2965149 {
			return true
		}
	case Chain_Holesky:
		// roughly 2024-08-01
		if blockNumber < 2046020 {
			return true
		}
	case Chain_Mainnet:
	}
	return false
}

func KebabToSnakeCase(str string) string {
	return strings.ReplaceAll(str, "-", "_")
}

func GetEnvVarVar(key string) string {
	// replace . with _ and uppercase
	key = strings.ToUpper(strings.ReplaceAll(key, ".", "_"))
	return fmt.Sprintf("%s_%s", ENV_PREFIX, key)
}

func StringVarToInt(val string) int {
	v, err := strconv.Atoi(val)
	if err != nil {
		return 0
	}
	return v
}
