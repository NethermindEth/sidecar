package eventBusTypes

import (
	"context"
	"github.com/Layr-Labs/sidecar/pkg/eigenState/stateManager"
	"github.com/Layr-Labs/sidecar/pkg/storage"
	"sync"
)

type Event struct {
	Name string
	Data any
}

type ConsumerId string

type Consumer struct {
	Id      ConsumerId
	Context context.Context
	Channel chan *Event
}

type ConsumerList struct {
	mu        sync.Mutex
	consumers []*Consumer
}

func NewConsumerList() *ConsumerList {
	return &ConsumerList{
		consumers: make([]*Consumer, 0),
	}
}

func (cl *ConsumerList) Add(consumer *Consumer) {
	cl.mu.Lock()
	defer cl.mu.Unlock()
	cl.consumers = append(cl.consumers, consumer)
}

func (cl *ConsumerList) Remove(consumer *Consumer) {
	cl.mu.Lock()
	defer cl.mu.Unlock()
	for i, c := range cl.consumers {
		if c.Id == consumer.Id {
			cl.consumers = append(cl.consumers[:i], cl.consumers[i+1:]...)
			break
		}
	}
}

func (cl *ConsumerList) GetAll() []*Consumer {
	cl.mu.Lock()
	defer cl.mu.Unlock()
	return cl.consumers
}

type IEventBus interface {
	Subscribe(consumer *Consumer)
	Unsubscribe(consumer *Consumer)
	Publish(event *Event)
}

type BlockProcessedData struct {
	Block          *storage.Block
	Transactions   []*storage.Transaction
	Logs           []*storage.TransactionLog
	StateRoot      *stateManager.StateRoot
	CommittedState map[string][]interface{}
}
