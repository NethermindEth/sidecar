package sidecar

import (
	"context"
	"github.com/Layr-Labs/sidecar/internal/clients/ethereum"
	"github.com/Layr-Labs/sidecar/internal/config"
	"github.com/Layr-Labs/sidecar/internal/pipeline"
	"github.com/Layr-Labs/sidecar/internal/storage"
	"go.uber.org/zap"
)

type SidecarConfig struct {
	GenesisBlockNumber uint64
}

type Sidecar struct {
	Logger         *zap.Logger
	Config         *SidecarConfig
	GlobalConfig   *config.Config
	Storage        storage.MetadataStore
	Pipeline       *pipeline.Pipeline
	EthereumClient *ethereum.Client
}

func NewSidecar(
	cfg *SidecarConfig,
	gCfg *config.Config,
	s storage.MetadataStore,
	p *pipeline.Pipeline,
	l *zap.Logger,
	ethClient *ethereum.Client,
) *Sidecar {
	return &Sidecar{
		Logger:         l,
		Config:         cfg,
		GlobalConfig:   gCfg,
		Storage:        s,
		Pipeline:       p,
		EthereumClient: ethClient,
	}
}

func (s *Sidecar) Start(ctx context.Context) {

	s.Logger.Info("Starting sidecar")

	s.StartIndexing(ctx)
	/*
		Main loop:

		- Get current indexed block
			- If no blocks, start from the genesis block
			- If some blocks, start from last indexed block
		- Once at tip, begin listening for new blocks
	*/
}
