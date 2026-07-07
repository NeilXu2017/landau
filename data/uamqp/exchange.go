package uamqp

import (
	"errors"
	"fmt"
	"math/rand"
	"net"
	"sync"
	"time"

	"github.com/NeilXu2017/landau/log"
	"github.com/streadway/amqp"
)

// NewExchange 构建Exchange
func NewExchange(nodes []NodeOptions, options ExchangeOptions) *Exchange {
	ex := &Exchange{
		Nodes:   nodes,
		options: options,
		Logger:  "main",
	}
	_ = ex.refreshConn()
	return ex
}

type (
	Exchange struct {
		m       sync.Mutex
		conn    *amqp.Connection
		Nodes   []NodeOptions
		options ExchangeOptions
		Logger  string
	}

	ExchangeOptions struct {
		Name       string
		Type       string
		Durable    bool
		AutoDelete bool
		Internal   bool
		NoWait     bool
		Arguments  amqp.Table
	}
)

// 打乱节点顺序
func (ex *Exchange) shuffleNodes() {
	n := len(ex.Nodes)
	if n <= 1 {
		return
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	i := n - 1
	for ; i > 1<<31-1-1; i-- {
		j := int(r.Int63n(int64(i + 1)))
		ex.Nodes[i], ex.Nodes[j] = ex.Nodes[j], ex.Nodes[i]
	}
	for ; i > 0; i-- {
		j := int(r.Int31n(int32(i + 1)))
		ex.Nodes[i], ex.Nodes[j] = ex.Nodes[j], ex.Nodes[i]
	}
}

func (ex *Exchange) refreshConn() error {
	ex.m.Lock()
	defer ex.m.Unlock()

	if ex.conn != nil {
		_ = ex.conn.Close()
	}
	if len(ex.Nodes) == 0 {
		return errors.New("get mq nodes failed")
	}
	// 每次重连时打乱节点顺序
	ex.shuffleNodes()
	for _, node := range ex.Nodes {
		conn, err := amqp.DialConfig(node.GetURI(), amqp.Config{
			Dial: func(network, addr string) (net.Conn, error) {
				return net.DialTimeout(network, addr, 3*time.Second)
			},
		})
		if err != nil {
			log.Error2(ex.Logger, "[AMQP Exchange] Dial error:%v", err)
			continue
		}
		ex.conn = conn
		return nil
	}
	return errors.New("not found alive node")
}

// NewChannel 创建新的Channel
func (ex *Exchange) NewChannel() (*amqp.Channel, error) {
	if ex.conn == nil {
		err := ex.refreshConn()

		if err != nil {
			return nil, err
		}
	}

	ex.m.Lock()
	defer ex.m.Unlock()

	ch, err := ex.conn.Channel()
	if err != nil {
		return nil, err
	}

	err = ch.ExchangeDeclare(
		ex.options.Name,
		ex.options.Type,
		ex.options.Durable,
		ex.options.AutoDelete,
		ex.options.Internal,
		ex.options.NoWait,
		ex.options.Arguments,
	)

	if err != nil {
		_ = ch.Close()
		return nil, fmt.Errorf("failed to declare an exchange: %s", err)
	}

	return ch, nil
}

// CreateConsumer 构建Consumer
func (ex *Exchange) CreateConsumer(routingKey string, consumerOptions ConsumerOptions, queueOptions QueueOptions) *Consumer {
	return NewConsumer(ex, routingKey, consumerOptions, queueOptions)
}

// CreateProducer 构建Producer
func (ex *Exchange) CreateProducer() *Producer {
	return NewProducer(ex)
}
