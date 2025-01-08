package eventBus

import (
	"github.com/Layr-Labs/sidecar/pkg/eventBus/eventBusTypes"
	"go.uber.org/zap"
	"sync"
)

type EventBus struct {
	consumers *eventBusTypes.ConsumerList
	logger    *zap.Logger
	lock      sync.Mutex
}

func NewEventBus(l *zap.Logger) *EventBus {
	return &EventBus{
		consumers: eventBusTypes.NewConsumerList(),
		logger:    l,
	}
}

func (eb *EventBus) Subscribe(consumer *eventBusTypes.Consumer) {
	eb.consumers.Add(consumer)
}

func (eb *EventBus) Unsubscribe(consumer *eventBusTypes.Consumer) {
	eb.consumers.Remove(consumer)
	eb.logger.Sugar().Infow("Unsubscribed consumer", zap.String("consumerId", string(consumer.Id)))
}

func (eb *EventBus) Publish(event *eventBusTypes.Event) {
	eb.logger.Sugar().Debugw("Publishing event", zap.String("eventName", event.Name))
	for _, consumer := range eb.consumers.GetAll() {
		if consumer.Channel != nil {
			select {
			case consumer.Channel <- event:
				eb.logger.Sugar().Debugw("Published event to consumer",
					zap.String("consumerId", string(consumer.Id)),
					zap.String("eventName", event.Name),
				)
			default:
				eb.logger.Sugar().Debugw("No receiver available, or channel is full",
					zap.String("consumerId", string(consumer.Id)),
					zap.String("eventName", event.Name),
				)
			}
		} else {
			eb.logger.Sugar().Debugw("Consumer channel is nil", zap.String("consumerId", string(consumer.Id)))
		}
	}
}
