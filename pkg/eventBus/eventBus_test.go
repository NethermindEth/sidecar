package eventBus

import (
	"context"
	"github.com/Layr-Labs/sidecar/internal/logger"
	"github.com/Layr-Labs/sidecar/pkg/eventBus/eventBusTypes"
	"github.com/stretchr/testify/assert"
	"sync"
	"sync/atomic"
	"testing"
)

func Test_EventBus(t *testing.T) {
	l, _ := logger.NewLogger(&logger.LoggerConfig{Debug: true})

	eb := NewEventBus(l)

	consumer := &eventBusTypes.Consumer{
		Id:      "testConsumer",
		Channel: make(chan *eventBusTypes.Event, 1000),
		Context: context.Background(),
	}

	receivedCount := atomic.Uint64{}
	receivedCount.Store(0)
	wg := sync.WaitGroup{}
	go func() {
		wg.Add(1)
		for {
			select {
			case event := <-consumer.Channel:
				t.Logf("Received event: %v", event)
				receivedCount.Add(1)
				
				if receivedCount.Load() == uint64(3) {
					eb.Unsubscribe(consumer)
					wg.Done()
					return
				}
			case <-consumer.Context.Done():
				return
			}
		}
	}()
	eb.Subscribe(consumer)

	for i := 0; i < 10; i++ {
		eb.Publish(&eventBusTypes.Event{
			Name: "testEvent",
			Data: "testData",
		})
	}
	wg.Wait()

	assert.Equal(t, uint64(2), receivedCount.Load())
}
