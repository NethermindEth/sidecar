package sequentialContractCaller

import (
	"context"
	"fmt"
	"github.com/Layr-Labs/go-sidecar/internal/config"
	"github.com/Layr-Labs/go-sidecar/internal/logger"
	"github.com/Layr-Labs/go-sidecar/internal/tests"
	"github.com/Layr-Labs/go-sidecar/pkg/clients/ethereum"
	"github.com/Layr-Labs/go-sidecar/pkg/utils"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"testing"
)

func setup() (
	*zap.Logger,
	*config.Config,
	error,
) {
	cfg := config.NewConfig()
	cfg.Chain = config.Chain_Mainnet
	cfg.EthereumRpcConfig.BaseUrl = "https://tame-fabled-liquid.quiknode.pro/f27d4be93b4d7de3679f5c5ae881233f857407a0"
	cfg.StatsdUrl = "localhost:8125"
	cfg.Debug = true
	cfg.DatabaseConfig = *tests.GetDbConfigFromEnv()

	l, _ := logger.NewLogger(&logger.LoggerConfig{Debug: true})

	return l, cfg, nil
}

func Test_SequentialContractCaller(t *testing.T) {
	l, cfg, err := setup()
	if err != nil {
		t.Fatal(err)
	}

	client := ethereum.NewClient(cfg.EthereumRpcConfig.BaseUrl, l)

	scc := NewSequentialContractCaller(client, cfg, l)

	t.Run("Get distribution root by index", func(t *testing.T) {
		distributionRoot, err := scc.GetDistributionRootByIndex(context.Background(), 8)
		if err != nil {
			t.Fatal(err)
		}

		rootString := utils.ConvertBytesToString(distributionRoot.Root[:])

		assert.Equal(t, distributionRoot.Disabled, true)
		assert.Equal(t, "0x8a7099a557f56bf18761cd2c303baeec71a60ee135107e7e02546dc547c16d99", rootString)

		fmt.Printf("Distribution root: %+v\n", distributionRoot)

	})
}
