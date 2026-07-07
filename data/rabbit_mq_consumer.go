package data

import (
	"sync"
	"time"

	"github.com/NeilXu2017/landau/data/uamqp"
	"github.com/NeilXu2017/landau/log"
	"github.com/streadway/amqp"
)

type (
	// RabbitMQConsumer Rabbit MQ 消息消费封装
	RabbitMQConsumer struct {
		logger               string
		nodes                []uamqp.NodeOptions
		exOptions            uamqp.ExchangeOptions
		consumerOptions      uamqp.ConsumerOptions
		queueOptions         uamqp.QueueOptions
		routingKey           string
		qosPrefetchCount     int
		qosPrefetchSize      int
		qosGlobal            bool
		consumer             *uamqp.Consumer
		sleepTime            time.Duration
		deliveryAutoAck      bool
		deliveryAutoAckValue bool
		deliveryAsync        bool
	}
	// RabbitMQConsumerOptionFunc 参数设置
	RabbitMQConsumerOptionFunc func(*RabbitMQConsumer) error
	// RabbitMQConsumerCallbackFunc 数据处理
	RabbitMQConsumerCallbackFunc func(*string, *amqp.Delivery)
)

const (
	// DefaultRabbitConsumerLogger default logger
	DefaultRabbitConsumerLogger = "main"
	// DefaultRabbitQosPrefetchCount default value
	DefaultRabbitQosPrefetchCount = 15
)

// NewRabbitConsumer RabbitMQConsumer
func NewRabbitConsumer(options ...RabbitMQConsumerOptionFunc) (*RabbitMQConsumer, error) {
	c := &RabbitMQConsumer{
		logger:               DefaultRabbitConsumerLogger,
		nodes:                []uamqp.NodeOptions{},
		exOptions:            uamqp.ExchangeOptions{Type: "topic", Durable: true},
		consumerOptions:      uamqp.ConsumerOptions{},
		queueOptions:         uamqp.QueueOptions{Durable: true},
		qosPrefetchCount:     DefaultRabbitQosPrefetchCount,
		sleepTime:            time.Second,
		deliveryAutoAck:      true,
		deliveryAutoAckValue: false,
		deliveryAsync:        true,
	}
	for _, option := range options {
		if err := option(c); err != nil {
			return nil, err
		}
	}
	mqEx := uamqp.NewExchange(c.nodes, c.exOptions)
	c.consumer = mqEx.CreateConsumer(c.routingKey, c.consumerOptions, c.queueOptions)
	c.consumer.SetQOS(c.qosPrefetchCount, c.qosPrefetchSize, c.qosGlobal)
	return c, nil
}

// StartListen 侦听消息
func (c *RabbitMQConsumer) StartListen(callback RabbitMQConsumerCallbackFunc, waitGroup *sync.WaitGroup) {
	defer func() {
		if waitGroup != nil {
			waitGroup.Done()
		}
	}()
	handle := func(delivery *amqp.Delivery) {
		message := string(delivery.Body)
		defer func() {
			if err := recover(); err != nil {
				log.Error2(c.logger, "[RabbitMQConsumer] Panic:%v Message:%s", err, message)
			}
			if c.deliveryAutoAck {
				_ = delivery.Ack(c.deliveryAutoAckValue)
			}
		}()
		log.Info2(c.logger, "[RabbitMQConsumer] [%s] receive message:%s", c.routingKey, message)
		callback(&message, delivery)
	}
	for {
		receiver, err := c.consumer.CreateReceiver()
		if err != nil {
			log.Error2(c.logger, "[RabbitMQConsumer] CreateReceiver error:%v", err)
			continue
		}
		log.Info2(c.logger, "[RabbitMQConsumer] CreateReceiver success, ExchangeName:%s QueueName:%s Consumer routingKey:%s,wait delivery message now...", c.exOptions.Name, c.queueOptions.Name, c.routingKey)
		for delivery := range receiver {
			if c.deliveryAsync {
				go handle(&delivery)
			} else {
				handle(&delivery)
			}
		}
		time.Sleep(c.sleepTime)
	}
}

// SetRabbitMQConsumerDeliveryAutoAck 设置 RabbitConsumer Delivery Ack
func SetRabbitMQConsumerDeliveryAutoAck(deliveryAutoAck bool) RabbitMQConsumerOptionFunc {
	return func(c *RabbitMQConsumer) error {
		c.deliveryAutoAck = deliveryAutoAck
		return nil
	}
}

// SetRabbitMQConsumerDeliveryAutoAckValue 设置 RabbitConsumer Delivery Ack Value
func SetRabbitMQConsumerDeliveryAutoAckValue(deliveryAutoAckValue bool) RabbitMQConsumerOptionFunc {
	return func(c *RabbitMQConsumer) error {
		c.deliveryAutoAckValue = deliveryAutoAckValue
		return nil
	}
}

// SetRabbitMQConsumerRetryConnectTime 设置 RabbitConsumer retry time after CreateReceiver error
func SetRabbitMQConsumerRetryConnectTime(sleepTime time.Duration) RabbitMQConsumerOptionFunc {
	return func(c *RabbitMQConsumer) error {
		c.sleepTime = sleepTime
		return nil
	}
}

// SetRabbitMQConsumerDeliveryAsync 设置 RabbitConsumer Delivery Ack Value
func SetRabbitMQConsumerDeliveryAsync(deliveryAsync bool) RabbitMQConsumerOptionFunc {
	return func(c *RabbitMQConsumer) error {
		c.deliveryAsync = deliveryAsync
		return nil
	}
}

// SetRabbitMQConsumerRoutingKey 设置 RabbitConsumer routingKey
func SetRabbitMQConsumerRoutingKey(routingKey string) RabbitMQConsumerOptionFunc {
	return func(c *RabbitMQConsumer) error {
		c.routingKey = routingKey
		return nil
	}
}

// SetRabbitMQConsumerQosPrefetchCount 设置 RabbitConsumer QOS  PrefetchCount
func SetRabbitMQConsumerQosPrefetchCount(prefetchCount int) RabbitMQConsumerOptionFunc {
	return func(c *RabbitMQConsumer) error {
		c.qosPrefetchCount = prefetchCount
		return nil
	}
}

// SetRabbitMQConsumerQosPrefetchSize 设置 RabbitConsumer QOS  PrefetchSize
func SetRabbitMQConsumerQosPrefetchSize(prefetchSize int) RabbitMQConsumerOptionFunc {
	return func(c *RabbitMQConsumer) error {
		c.qosPrefetchSize = prefetchSize
		return nil
	}
}

// SetRabbitMQConsumerQosGlobal 设置 RabbitConsumer QOS  Global
func SetRabbitMQConsumerQosGlobal(global bool) RabbitMQConsumerOptionFunc {
	return func(c *RabbitMQConsumer) error {
		c.qosGlobal = global
		return nil
	}
}

// SetRabbitMQConsumerNode 设置 node
func SetRabbitMQConsumerNode(host string, port int, user string, password string) RabbitMQConsumerOptionFunc {
	return func(c *RabbitMQConsumer) error {
		c.nodes = append(c.nodes, uamqp.NodeOptions{Host: host, Port: port, User: user, Password: password})
		return nil
	}
}

// SetRabbitMQConsumerExchangeName  设置 ExchangeOptions Name
func SetRabbitMQConsumerExchangeName(name string) RabbitMQConsumerOptionFunc {
	return func(c *RabbitMQConsumer) error {
		c.exOptions.Name = name
		return nil
	}
}

// SetRabbitMQConsumerExchangeType 设置 ExchangeOptions Type
func SetRabbitMQConsumerExchangeType(exchangeType string) RabbitMQConsumerOptionFunc {
	return func(c *RabbitMQConsumer) error {
		c.exOptions.Type = exchangeType
		return nil
	}
}

// SetRabbitMQConsumerExchangeDurable 设置 ExchangeOptions Durable
func SetRabbitMQConsumerExchangeDurable(durable bool) RabbitMQConsumerOptionFunc {
	return func(c *RabbitMQConsumer) error {
		c.exOptions.Durable = durable
		return nil
	}
}

// SetRabbitMQConsumerExchangeAutoDelete 设置 ExchangeOptions AutoDelete
func SetRabbitMQConsumerExchangeAutoDelete(autoDelete bool) RabbitMQConsumerOptionFunc {
	return func(c *RabbitMQConsumer) error {
		c.exOptions.AutoDelete = autoDelete
		return nil
	}
}

// SetRabbitMQConsumerExchangeInternal 设置 ExchangeOptions Internal
func SetRabbitMQConsumerExchangeInternal(internal bool) RabbitMQConsumerOptionFunc {
	return func(c *RabbitMQConsumer) error {
		c.exOptions.Internal = internal
		return nil
	}
}

// SetRabbitMQConsumerExchangeNoWait 设置 ExchangeOptions NoWait
func SetRabbitMQConsumerExchangeNoWait(noWait bool) RabbitMQConsumerOptionFunc {
	return func(c *RabbitMQConsumer) error {
		c.exOptions.NoWait = noWait
		return nil
	}
}

// SetRabbitMQConsumerExchangeArguments  设置 ExchangeOptions Arguments
func SetRabbitMQConsumerExchangeArguments(arguments amqp.Table) RabbitMQConsumerOptionFunc {
	return func(c *RabbitMQConsumer) error {
		c.exOptions.Arguments = arguments
		return nil
	}
}

// SetRabbitMQConsumerName 设置 ConsumerOptions Name
func SetRabbitMQConsumerName(name string) RabbitMQConsumerOptionFunc {
	return func(c *RabbitMQConsumer) error {
		c.consumerOptions.Name = name
		return nil
	}
}

// SetRabbitMQConsumerTag 设置 ConsumerOptions Tag
func SetRabbitMQConsumerTag(tag string) RabbitMQConsumerOptionFunc {
	return func(c *RabbitMQConsumer) error {
		c.consumerOptions.ConsumerTag = tag
		return nil
	}
}

// SetRabbitMQConsumerNoLocal 设置 ConsumerOptions NoLocal
func SetRabbitMQConsumerNoLocal(noLocal bool) RabbitMQConsumerOptionFunc {
	return func(c *RabbitMQConsumer) error {
		c.consumerOptions.NoLocal = noLocal
		return nil
	}
}

// SetRabbitMQConsumerAutoAck 设置 ConsumerOptions AutoAck
func SetRabbitMQConsumerAutoAck(autoAck bool) RabbitMQConsumerOptionFunc {
	return func(c *RabbitMQConsumer) error {
		c.consumerOptions.AutoAck = autoAck
		return nil
	}
}

// SetRabbitMQConsumerExclusive 设置 ConsumerOptions Exclusive
func SetRabbitMQConsumerExclusive(exclusive bool) RabbitMQConsumerOptionFunc {
	return func(c *RabbitMQConsumer) error {
		c.consumerOptions.Exclusive = exclusive
		return nil
	}
}

// SetRabbitMQConsumerNoWait 设置 ConsumerOptions NoWait
func SetRabbitMQConsumerNoWait(noWait bool) RabbitMQConsumerOptionFunc {
	return func(c *RabbitMQConsumer) error {
		c.consumerOptions.NoWait = noWait
		return nil
	}
}

// SetRabbitMQConsumerArguments 设置 ConsumerOptions Arguments
func SetRabbitMQConsumerArguments(arguments amqp.Table) RabbitMQConsumerOptionFunc {
	return func(c *RabbitMQConsumer) error {
		c.consumerOptions.Arguments = arguments
		return nil
	}
}

// SetRabbitMQConsumerQueueName 设置 QueueOptions Name
func SetRabbitMQConsumerQueueName(name string) RabbitMQConsumerOptionFunc {
	return func(c *RabbitMQConsumer) error {
		c.queueOptions.Name = name
		return nil
	}
}

// SetRabbitMQConsumerQueueDurable 设置 QueueOptions Durable
func SetRabbitMQConsumerQueueDurable(durable bool) RabbitMQConsumerOptionFunc {
	return func(c *RabbitMQConsumer) error {
		c.queueOptions.Durable = durable
		return nil
	}
}

// SetRabbitMQConsumerQueueAutoDelete 设置 QueueOptions AutoDelete
func SetRabbitMQConsumerQueueAutoDelete(autoDelete bool) RabbitMQConsumerOptionFunc {
	return func(c *RabbitMQConsumer) error {
		c.queueOptions.AutoDelete = autoDelete
		return nil
	}
}

// SetRabbitMQConsumerQueueExclusive 设置 QueueOptions Exclusive
func SetRabbitMQConsumerQueueExclusive(exclusive bool) RabbitMQConsumerOptionFunc {
	return func(c *RabbitMQConsumer) error {
		c.queueOptions.Exclusive = exclusive
		return nil
	}
}

// SetRabbitMQConsumerQueueNoWait 设置 QueueOptions NoWait
func SetRabbitMQConsumerQueueNoWait(noWait bool) RabbitMQConsumerOptionFunc {
	return func(c *RabbitMQConsumer) error {
		c.queueOptions.NoWait = noWait
		return nil
	}
}

// SetRabbitMQConsumerQueueArguments 设置 QueueOptions Arguments
func SetRabbitMQConsumerQueueArguments(arguments amqp.Table) RabbitMQConsumerOptionFunc {
	return func(c *RabbitMQConsumer) error {
		c.queueOptions.Arguments = arguments
		return nil
	}
}
