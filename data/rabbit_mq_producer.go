package data

import (
	"github.com/NeilXu2017/landau/data/uamqp"
	"github.com/NeilXu2017/landau/log"
	"github.com/streadway/amqp"
)

type (
	// RabbitMQProducer Rabbit MQ 消息生成封装
	RabbitMQProducer struct {
		logger     string
		nodes      []uamqp.NodeOptions
		exOptions  uamqp.ExchangeOptions
		routingKey string
		producer   *uamqp.Producer
	}
	// RabbitMQProducerOptionFunc 参数设置
	RabbitMQProducerOptionFunc func(*RabbitMQProducer) error
	// RabbitMQProducerPublishOptionFunc Publish 参数设置
	RabbitMQProducerPublishOptionFunc func(*amqp.Publishing) error
)

const (
	// DefaultRabbitProducerLogger default logger
	DefaultRabbitProducerLogger = "main"
	// DefaultContentType default content type
	DefaultContentType = "application/json"
	// DefaultContentEncoding default content encoding
	DefaultContentEncoding = "utf-8"
	// DefaultPriority default priority
	DefaultPriority = 0
	// DefaultDeliveryMode default deliver mode
	DefaultDeliveryMode = amqp.Persistent
)

// NewRabbitProducer 构建RabbitProducer
func NewRabbitProducer(options ...RabbitMQProducerOptionFunc) (*RabbitMQProducer, error) {
	c := &RabbitMQProducer{
		logger:     DefaultRabbitProducerLogger,
		nodes:      []uamqp.NodeOptions{},
		exOptions:  uamqp.ExchangeOptions{Type: "topic"},
		routingKey: "",
	}
	for _, option := range options {
		if err := option(c); err != nil {
			return nil, err
		}
	}
	mqEx := uamqp.NewExchange(c.nodes, c.exOptions)
	c.producer = uamqp.NewProducer(mqEx)
	return c, nil
}

// Publish 产生消息
func (c *RabbitMQProducer) Publish(message string, mandatory, immediate bool, options ...RabbitMQProducerPublishOptionFunc) error {
	publishing := amqp.Publishing{
		Headers:         amqp.Table{},
		ContentType:     DefaultContentType,
		ContentEncoding: DefaultContentEncoding,
		Body:            []byte(message),
		DeliveryMode:    DefaultDeliveryMode,
		Priority:        DefaultPriority,
	}
	for _, option := range options {
		if err := option(&publishing); err != nil {
			return err
		}
	}
	err := c.producer.Publish(c.routingKey, publishing, mandatory, immediate)
	if err != nil {
		log.Error2(c.logger, "[RabbitMQProducer]\tPublish (routingKey:%s %v) error:%v", c.routingKey, publishing, err)
	}
	return err
}

// SetRabbitMQProducerPublishContentType Publish content type
func SetRabbitMQProducerPublishContentType(contentType string) RabbitMQProducerPublishOptionFunc {
	return func(c *amqp.Publishing) error {
		c.ContentType = contentType
		return nil
	}
}

// SetRabbitMQProducerPublishContentEncoding Publish content encoding
func SetRabbitMQProducerPublishContentEncoding(contentEncoding string) RabbitMQProducerPublishOptionFunc {
	return func(c *amqp.Publishing) error {
		c.ContentEncoding = contentEncoding
		return nil
	}
}

// SetRabbitMQProducerPublishDeliverMode Publish deliver mode
func SetRabbitMQProducerPublishDeliverMode(deliverMode uint8) RabbitMQProducerPublishOptionFunc {
	return func(c *amqp.Publishing) error {
		c.DeliveryMode = deliverMode
		return nil
	}
}

// SetRabbitMQProducerPublishPriority Publish priority
func SetRabbitMQProducerPublishPriority(priority uint8) RabbitMQProducerPublishOptionFunc {
	return func(c *amqp.Publishing) error {
		c.Priority = priority
		return nil
	}
}

// SetRabbitMQProducerPublishHeaders Publish Headers
func SetRabbitMQProducerPublishHeaders(headers amqp.Table) RabbitMQProducerPublishOptionFunc {
	return func(c *amqp.Publishing) error {
		c.Headers = headers
		return nil
	}
}

// SetRabbitMQProducerLogger 设置 RabbitProducer logger
func SetRabbitMQProducerLogger(logger string) RabbitMQProducerOptionFunc {
	return func(c *RabbitMQProducer) error {
		c.logger = logger
		return nil
	}
}

// SetRabbitMQProducerRoutingKey 设置 RabbitProducer routingKey
func SetRabbitMQProducerRoutingKey(routingKey string) RabbitMQProducerOptionFunc {
	return func(c *RabbitMQProducer) error {
		c.routingKey = routingKey
		return nil
	}
}

// SetRabbitMQProducerNode 设置 node
func SetRabbitMQProducerNode(host string, port int, user string, password string) RabbitMQProducerOptionFunc {
	return func(c *RabbitMQProducer) error {
		c.nodes = append(c.nodes, uamqp.NodeOptions{Host: host, Port: port, User: user, Password: password})
		return nil
	}
}

// SetRabbitMQProducerExchangeName  设置 ExchangeOptions Name
func SetRabbitMQProducerExchangeName(name string) RabbitMQProducerOptionFunc {
	return func(c *RabbitMQProducer) error {
		c.exOptions.Name = name
		return nil
	}
}

// SetRabbitMQProducerExchangeType 设置 ExchangeOptions Type
func SetRabbitMQProducerExchangeType(exchangeType string) RabbitMQProducerOptionFunc {
	return func(c *RabbitMQProducer) error {
		c.exOptions.Type = exchangeType
		return nil
	}
}

// SetRabbitMQProducerExchangeDurable 设置 ExchangeOptions Durable
func SetRabbitMQProducerExchangeDurable(durable bool) RabbitMQProducerOptionFunc {
	return func(c *RabbitMQProducer) error {
		c.exOptions.Durable = durable
		return nil
	}
}

// SetRabbitMQProducerExchangeAutoDelete 设置 ExchangeOptions AutoDelete
func SetRabbitMQProducerExchangeAutoDelete(autoDelete bool) RabbitMQProducerOptionFunc {
	return func(c *RabbitMQProducer) error {
		c.exOptions.AutoDelete = autoDelete
		return nil
	}
}

// SetRabbitMQProducerExchangeInternal 设置 ExchangeOptions Internal
func SetRabbitMQProducerExchangeInternal(internal bool) RabbitMQProducerOptionFunc {
	return func(c *RabbitMQProducer) error {
		c.exOptions.Internal = internal
		return nil
	}
}

// SetRabbitMQProducerExchangeNoWait 设置 ExchangeOptions NoWait
func SetRabbitMQProducerExchangeNoWait(noWait bool) RabbitMQProducerOptionFunc {
	return func(c *RabbitMQProducer) error {
		c.exOptions.NoWait = noWait
		return nil
	}
}

// SetRabbitMQProducerExchangeArguments  设置 ExchangeOptions Arguments
func SetRabbitMQProducerExchangeArguments(arguments amqp.Table) RabbitMQProducerOptionFunc {
	return func(c *RabbitMQProducer) error {
		c.exOptions.Arguments = arguments
		return nil
	}
}
