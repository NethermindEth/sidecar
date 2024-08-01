package rabbitmq

import (
	"fmt"
	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

type RabbitMQQueue struct {
	Name           string
	Durable        bool
	AutoAck        bool
	Exclusive      bool
	BindExchange   string
	BindRoutingKey string
}

type RabbitMQExchange struct {
	Name       string
	Durable    bool
	AutoDelete bool
	Kind       string
}

type RabbitMQConfig struct {
	Username  string
	Password  string
	Url       string
	Secure    bool
	Queues    []*RabbitMQQueue
	Exchanges []*RabbitMQExchange
}

type RabbitMQ struct {
	logger     *zap.Logger
	config     *RabbitMQConfig
	connection *amqp.Connection
	channel    *amqp.Channel
}

func NewRabbitMQ(config *RabbitMQConfig, l *zap.Logger) *RabbitMQ {
	return &RabbitMQ{
		config: config,
		logger: l,
	}
}

func (r *RabbitMQ) Connect() (*amqp.Connection, error) {
	connUrl := buildConnectionUrl(r.config)
	r.logger.Sugar().Debug(fmt.Sprintf("Connecting to RabbitMQ at %s", connUrl))
	conn, err := amqp.Dial(connUrl)
	if err != nil {
		r.logger.Sugar().Errorw(fmt.Sprintf("Failed to connect to RabbitMQ: %v", err))
		return nil, err
	}
	r.connection = conn

	ch, err := conn.Channel()
	if err != nil {
		r.logger.Sugar().Errorw(fmt.Sprintf("Failed to open a channel: %v", err))
		return nil, err
	}
	r.channel = ch

	for _, e := range r.config.Exchanges {
		r.logger.Sugar().Debug(fmt.Sprintf("Declaring exchange %s", e.Name))
		err = r.channel.ExchangeDeclare(e.Name, e.Kind, e.Durable, e.AutoDelete, false, false, nil)
		if err != nil {
			return nil, err
		}
	}

	for _, q := range r.config.Queues {
		r.logger.Sugar().Debug(fmt.Sprintf("Declaring queue %s", q.Name))
		_, err = r.channel.QueueDeclare(q.Name, q.Durable, false, false, false, nil)
		if err != nil {
			return nil, err
		}

		if q.BindExchange != "" {
			r.logger.Sugar().Debug(fmt.Sprintf("Binding queue %s to exchange %s with routing key %s", q.Name, q.BindExchange, q.BindRoutingKey))
			err = r.channel.QueueBind(q.Name, q.BindRoutingKey, q.BindExchange, false, nil)
			if err != nil {
				return nil, err
			}
		}

	}

	return conn, nil
}

func (r *RabbitMQ) SetQos(prefetchCount int) error {
	return r.channel.Qos(prefetchCount, 0, false)
}

func (r *RabbitMQ) Publish(exchangeName string, routingKey string, publishing amqp.Publishing) error {
	return r.channel.Publish(exchangeName, routingKey, false, false, publishing)
}

func (r *RabbitMQ) Consume(queueName, consumerName string, autoAck, exclusive, noLocal, noWait bool, args amqp.Table) (<-chan amqp.Delivery, error) {
	return r.channel.Consume(queueName, consumerName, autoAck, exclusive, noLocal, noWait, args)
}

func (r *RabbitMQ) PurgeAllQueues() {
	for _, q := range r.config.Queues {
		r.PurgeQueue(q.Name, false)
	}
}

func (r *RabbitMQ) PurgeQueue(queueName string, noWait bool) (int, error) {
	return r.channel.QueuePurge(queueName, noWait)
}

func buildConnectionUrl(cfg *RabbitMQConfig) string {
	protocol := "amqp"
	if cfg.Secure {
		protocol = "amqps"
	}
	return fmt.Sprintf("%s://%s:%s@%s", protocol, cfg.Username, cfg.Password, cfg.Url)
}
