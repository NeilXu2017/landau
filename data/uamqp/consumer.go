package uamqp

import (
	"sync"

	"github.com/streadway/amqp"
)

func NewConsumer(exchange *Exchange, routingKey string, consumerOptions ConsumerOptions, queueOptions QueueOptions) *Consumer {
	return &Consumer{
		Exchange:     exchange,
		routingKey:   routingKey,
		options:      consumerOptions,
		queueOptions: queueOptions,
	}
}

type (
	Consumer struct {
		m       sync.Mutex
		Channel *amqp.Channel

		Exchange   *Exchange
		routingKey string

		options      ConsumerOptions
		queueOptions QueueOptions

		qosOptions *QOSOptions
	}
	ConsumerOptions struct {
		Name        string
		ConsumerTag string
		NoLocal     bool
		AutoAck     bool
		Exclusive   bool
		NoWait      bool
		Arguments   amqp.Table
	}
	QueueOptions struct {
		Name       string
		Durable    bool
		AutoDelete bool
		Exclusive  bool
		NoWait     bool
		Arguments  amqp.Table
	}

	QOSOptions struct {
		prefetchCount int
		prefetchSize  int
		global        bool
	}
)

func (c *Consumer) SetQOS(prefetchCount int, prefetchSize int, global bool) *Consumer {
	c.qosOptions = &QOSOptions{prefetchCount, prefetchSize, global}
	return c
}

// CreateReceiver 创建一个<-chan amqp.Delivery类型的通道用于接收数据
func (c *Consumer) CreateReceiver() (<-chan amqp.Delivery, error) {
	_, err := c.getChannel()
	if err != nil {
		c.CloseForErr(err)
		return nil, err
	}

	if c.qosOptions != nil {
		err := c.Channel.Qos(c.qosOptions.prefetchCount, c.qosOptions.prefetchSize, c.qosOptions.global)
		if err != nil {
			c.CloseForErr(err)
			return nil, err
		}
	}

	receiver, err := c.Channel.Consume(
		c.queueOptions.Name,
		c.options.ConsumerTag,
		c.options.AutoAck,
		c.options.Exclusive,
		c.options.NoLocal,
		c.options.NoWait,
		c.options.Arguments,
	)
	if err != nil {
		c.CloseForErr(err)
		return nil, err
	}

	return receiver, nil
}

// 为当前Consumer创建一个Channel
func (c *Consumer) getChannel() (*amqp.Channel, error) {
	c.m.Lock()
	defer c.m.Unlock()

	if c.Channel != nil {
		return c.Channel, nil
	}

	var err error
	c.Channel, err = c.Exchange.NewChannel()
	if err != nil {
		return nil, err
	}

	_, err = c.Channel.QueueDeclare(
		c.queueOptions.Name,
		c.queueOptions.Durable,
		c.queueOptions.AutoDelete,
		c.queueOptions.Exclusive,
		c.queueOptions.NoWait,
		c.queueOptions.Arguments,
	)
	if err != nil {
		return c.Channel, err
	}

	err = c.Channel.QueueBind(c.queueOptions.Name, c.routingKey, c.Exchange.options.Name, false, nil)
	if err != nil {
		return c.Channel, err
	}

	return c.Channel, nil
}

// CloseForErr 关闭channel, 如果链接断开则重连
func (c *Consumer) CloseForErr(oriErr error) {
	c.m.Lock()
	defer c.m.Unlock()

	if c.Channel != nil {
		_ = c.Channel.Cancel(c.options.ConsumerTag, false)
		_ = c.Channel.Close()
		c.Channel = nil
	}

	if oriErr == amqp.ErrClosed {
		_ = c.Exchange.refreshConn()
	}
}
