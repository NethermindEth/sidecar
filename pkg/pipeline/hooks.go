package pipeline

import (
	"github.com/Layr-Labs/sidecar/pkg/eigenState/stateManager"
	"github.com/Layr-Labs/sidecar/pkg/eventBus/eventBusTypes"
	"github.com/Layr-Labs/sidecar/pkg/storage"
)

func (p *Pipeline) HandleBlockProcessedHook(
	block *storage.Block,
	transactions []*storage.Transaction,
	logs []*storage.TransactionLog,
	stateRoot *stateManager.StateRoot,
	committedState map[string][]interface{},
) {
	p.eventBus.Publish(&eventBusTypes.Event{
		Name: eventBusTypes.Event_BlockProcessed,
		Data: &eventBusTypes.BlockProcessedData{
			Block:          block,
			Transactions:   transactions,
			Logs:           logs,
			StateRoot:      stateRoot,
			CommittedState: committedState,
		},
	})
}
