package sidecar

import (
	"context"
	"go.uber.org/zap"
	"sync"
	"time"
)

func (s *Sidecar) GetLastIndexedBlock() (int64, error) {
	block, err := s.Storage.GetLatestBlock()
	if err != nil {
		s.Logger.Sugar().Errorw("Failed to get last indexed block", zap.Error(err))
	}
	return int64(block.Number), nil
}

func (s *Sidecar) StartIndexing(ctx context.Context) {
	// Start indexing from the given block number
	// Once at tip, begin listening for new blocks
	if err := s.IndexFromCurrentToTip(ctx); err != nil {
		s.Logger.Sugar().Fatalw("Failed to index from current to tip", zap.Error(err))
	}
}

type currentTip struct {
	sync.RWMutex
	CurrentTip uint64
}

func (ct *currentTip) Get() uint64 {
	ct.RLock()
	defer ct.RUnlock()
	return ct.CurrentTip
}

func (ct *currentTip) Set(tip uint64) {
	ct.Lock()
	defer ct.Unlock()
	ct.CurrentTip = tip
}

func (s *Sidecar) IndexFromCurrentToTip(ctx context.Context) error {
	latestBlock, err := s.GetLastIndexedBlock()
	if err != nil {
		return err
	}
	if latestBlock == 0 {
		s.Logger.Sugar().Infow("No blocks indexed, starting from genesis block", zap.Uint64("genesisBlock", s.Config.GenesisBlockNumber))
		latestBlock = int64(s.Config.GenesisBlockNumber)
	}

	blockNumber, err := s.EthereumClient.GetBlockNumberUint64(ctx)
	if err != nil {
		s.Logger.Sugar().Fatalw("Failed to get current tip", zap.Error(err))
	}

	s.Logger.Sugar().Infow("Indexing from current to tip",
		zap.Uint64("currentTip", blockNumber),
		zap.Int64("latestBlock", latestBlock),
		zap.Uint64("difference", blockNumber-uint64(latestBlock)),
	)

	ct := currentTip{CurrentTip: blockNumber}

	go func() {
		for {
			time.Sleep(time.Second * 30)
			latestTip, err := s.EthereumClient.GetBlockNumberUint64(ctx)
			if err != nil {
				s.Logger.Sugar().Errorw("Failed to get latest tip", zap.Error(err))
				continue
			}
			if latestTip > ct.Get() {
				s.Logger.Sugar().Infow("New tip found, updating",
					zap.Uint64("latestTip", latestTip),
					zap.Uint64("ct", ct.Get()),
				)
				ct.Set(latestTip)
			}
		}
	}()
	for i := latestBlock; i <= int64(ct.Get()); i++ {
		if err := s.Pipeline.RunForBlock(ctx, uint64(i)); err != nil {
			s.Logger.Sugar().Errorw("Failed to run pipeline for block",
				zap.Int64("currentBlockNumber", i),
				zap.Error(err),
			)
			return err
		}
	}
	return nil
}
