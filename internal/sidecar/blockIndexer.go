package sidecar

import "go.uber.org/zap"

func (s *Sidecar) GetLastIndexedBlock() (int64, error) {
	block, err := s.Storage.GetLatestBlock()
	if err != nil {
		s.Logger.Sugar().Errorw("Failed to get last indexed block", zap.Error(err))
	}
	return block, nil
}

func (s *Sidecar) StartIndexing(startBlockNumber uint64) {
	// Start indexing from the given block number
	// Once at tip, begin listening for new blocks
}

func (s *Sidecar) HandleBlock(blockNumber uint64) {

}
