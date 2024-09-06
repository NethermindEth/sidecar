package eigenState

import (
	"encoding/json"
	"fmt"
	"github.com/Layr-Labs/sidecar/internal/parser"
	"github.com/Layr-Labs/sidecar/internal/storage"
	"go.uber.org/zap"
)

type BaseEigenState struct {
	Logger *zap.Logger
}

func (b *BaseEigenState) ParseLogArguments(log *storage.TransactionLog) ([]parser.Argument, error) {
	arguments := make([]parser.Argument, 0)
	err := json.Unmarshal([]byte(log.Arguments), &arguments)
	if err != nil {
		b.Logger.Sugar().Errorw("Failed to unmarshal arguments",
			zap.Error(err),
			zap.String("transactionHash", log.TransactionHash),
			zap.Uint64("transactionIndex", log.TransactionIndex),
		)
		return nil, err
	}
	return arguments, nil
}

func (b *BaseEigenState) ParseLogOutput(log *storage.TransactionLog) (map[string]interface{}, error) {
	outputData := make(map[string]interface{})
	err := json.Unmarshal([]byte(log.OutputData), &outputData)
	if err != nil {
		b.Logger.Sugar().Errorw("Failed to unmarshal outputData",
			zap.Error(err),
			zap.String("transactionHash", log.TransactionHash),
			zap.Uint64("transactionIndex", log.TransactionIndex),
		)
		return nil, err
	}
	return outputData, nil
}

// Include the block number as the first item in the tree.
// This does two things:
// 1. Ensures that the tree is always different for different blocks
// 2. Allows us to have at least 1 value if there are no model changes for a block
func (b *BaseEigenState) InitializeMerkleTreeBaseStateWithBlock(blockNumber uint64) [][]byte {
	return [][]byte{
		[]byte(fmt.Sprintf("%d", blockNumber)),
	}
}
