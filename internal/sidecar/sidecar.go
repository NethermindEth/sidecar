package sidecar

import (
	"github.com/Layr-Labs/sidecar/internal/config"
	"github.com/Layr-Labs/sidecar/internal/pipeline"
	"github.com/Layr-Labs/sidecar/internal/storage"
	"go.uber.org/zap"
)

type SidecarConfig struct {
	GenesisBlockNumber int64
}

type Sidecar struct {
	Logger       *zap.Logger
	Config       *SidecarConfig
	GlobalConfig *config.Config
	Storage      storage.MetadataStore
	Pipeline     *pipeline.Pipeline
}

func NewSidecar(
	l *zap.Logger,
	cfg *SidecarConfig,
	gCfg *config.Config,
	s storage.MetadataStore,
	p *pipeline.Pipeline,
) *Sidecar {
	return &Sidecar{
		Logger:       l,
		Config:       cfg,
		GlobalConfig: gCfg,
		Storage:      s,
		Pipeline:     p,
	}
}

func (s *Sidecar) Start() {
	s.Logger.Info("Starting sidecar")

	latestBlock, err := s.GetLastIndexedBlock()
	if err != nil {
		s.Logger.Sugar().Fatalw("Failed to get last indexed block", zap.Error(err))
	}

	if latestBlock == -1 {
		latestBlock = s.Config.GenesisBlockNumber
	}
	s.StartIndexing(uint64(latestBlock))
	/*
		Main loop:

		- Get current indexed block
			- If no blocks, start from the genesis block
			- If some blocks, start from last indexed block
		- Once at tip, begin listening for new blocks
	*/
}
