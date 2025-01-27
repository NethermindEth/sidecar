package baseDataService

import (
	"context"
	"github.com/Layr-Labs/sidecar/pkg/storage"
	"gorm.io/gorm"
)

type BaseDataService struct {
	DB *gorm.DB
}

func (b *BaseDataService) GetCurrentBlockHeightIfNotPresent(ctx context.Context, blockHeight uint64) (uint64, error) {
	if blockHeight == 0 {
		var currentBlock *storage.Block
		res := b.DB.Model(&storage.Block{}).Order("number desc").First(&currentBlock)
		if res.Error != nil {
			return 0, res.Error
		}
		blockHeight = currentBlock.Number
	}
	return blockHeight, nil
}
