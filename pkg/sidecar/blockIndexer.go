package sidecar

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
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

	latestStateRoot, err := s.StateManager.GetLatestStateRoot()
	if err != nil {
		s.Logger.Sugar().Errorw("Failed to get latest state root", zap.Error(err))
		return err
	}

	if latestStateRoot != nil {
		s.Logger.Sugar().Infow("Comparing latest block and latest state root",
			zap.Int64("latestBlock", latestBlock),
			zap.Uint64("latestStateRootBlock", latestStateRoot.EthBlockNumber),
		)
	} else {
		s.Logger.Sugar().Infow("No state roots found, starting from EL genesis")
		latestBlock = 0
	}

	if latestBlock == 0 {
		s.Logger.Sugar().Infow("No blocks indexed, starting from genesis block", zap.Uint64("genesisBlock", s.Config.GenesisBlockNumber))
		latestBlock = int64(s.Config.GenesisBlockNumber)
	} else {
		// if the latest state root is behind the latest block, delete the corrupted state and set the
		// latest block to the latest state root + 1
		if latestStateRoot != nil && latestStateRoot.EthBlockNumber < uint64(latestBlock) {
			s.Logger.Sugar().Infow("Latest state root is behind latest block, deleting corrupted state",
				zap.Uint64("latestStateRoot", latestStateRoot.EthBlockNumber),
				zap.Int64("latestBlock", latestBlock),
			)
			if err := s.StateManager.DeleteCorruptedState(latestStateRoot.EthBlockNumber+1, uint64(latestBlock)); err != nil {
				s.Logger.Sugar().Errorw("Failed to delete corrupted state", zap.Error(err))
				return err
			}
			if err := s.Storage.DeleteCorruptedState(uint64(latestStateRoot.EthBlockNumber+1), uint64(latestBlock)); err != nil {
				s.Logger.Sugar().Errorw("Failed to delete corrupted state", zap.Error(err))
				return err
			}
		} else {
			// otherwise, start from the latest block + 1
			latestBlock += 1
		}
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

	// Spin up a goroutine that listens on a channel for a shutdown signal.
	// When the signal is received, set shouldShutdown to true and return.
	go func() {
		for range s.ShutdownChan {
			s.Logger.Sugar().Infow("Received shutdown signal")
			shouldShutdown = true
		}
	}()

	// Every 30 seconds, check to see if the current tip has changed while the backfill/sync
	// process is still running. If it has changed, update the value which will extend the loop
	// to include the newly discovered blocks.
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
	// Keep some metrics during the indexing process
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
