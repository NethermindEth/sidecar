package sidecar

import (
	"context"
	"fmt"
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
	} else {
		latestBlock += 1
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

	shouldShutdown := false

	go func() {
		for {
			select {
			case <-s.ShutdownChan:
				s.Logger.Sugar().Infow("Received shutdown signal")
				shouldShutdown = true
			}
		}
	}()

	go func() {
		for {
			time.Sleep(time.Second * 30)
			if shouldShutdown {
				s.Logger.Sugar().Infow("Shutting down block listener...")
				return
			}
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
	blocksProcessed := 0
	runningAvg := 0
	totalDurationMs := 0
	lastBlockParsed := latestBlock
	for i := latestBlock; i <= int64(ct.Get()); i++ {
		if shouldShutdown {
			s.Logger.Sugar().Infow("Shutting down block processor")
			return nil
		}
		tip := ct.Get()
		blocksRemaining := tip - uint64(i)
		pctComplete := (float64(blocksProcessed) / float64(blocksRemaining)) * 100
		estTimeRemainingMs := runningAvg * int(blocksRemaining)
		estTimeRemainingHours := float64(estTimeRemainingMs) / 1000 / 60 / 60

		if i%10 == 0 {
			s.Logger.Sugar().Infow("Progress",
				zap.String("percentComplete", fmt.Sprintf("%.2f", pctComplete)),
				zap.Uint64("blocksRemaining", blocksRemaining),
				zap.Float64("estimatedTimeRemaining (hrs)", estTimeRemainingHours),
				zap.Float64("avgBlockProcessTime (ms)", float64(runningAvg)),
				zap.Uint64("lastBlockParsed", uint64(lastBlockParsed)),
			)
		}

		startTime := time.Now()
		if err := s.Pipeline.RunForBlock(ctx, uint64(i)); err != nil {
			s.Logger.Sugar().Errorw("Failed to run pipeline for block",
				zap.Int64("currentBlockNumber", i),
				zap.Error(err),
			)
			return err
		}

		lastBlockParsed = i
		delta := time.Since(startTime).Milliseconds()
		blocksProcessed++

		totalDurationMs += int(delta)
		runningAvg = totalDurationMs / blocksProcessed

		s.Logger.Sugar().Debugw("Processed block",
			zap.Int64("blockNumber", i),
			zap.Int64("duration", delta),
			zap.Int("avgDuration", runningAvg),
		)
	}

	// TODO(seanmcgary): transition to listening for new blocks
	return nil
}
