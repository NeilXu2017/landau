package uamqp

import (
	"fmt"
	"sync"

	"github.com/streadway/amqp"
)

func NewProducer(exchange *Exchange) *Producer {
	return &Producer{
		Exchange: exchange,
	}
}

type Producer struct {
	m       sync.Mutex
	Channel *amqp.Channel

	Exchange *Exchange
}

func (p *Producer) getChannel() (*amqp.Channel, error) {
	p.m.Lock()
	defer p.m.Unlock()

	if p.Channel != nil {
		return p.Channel, nil
	}

	var err error
	p.Channel, err = p.Exchange.NewChannel()
	if err != nil {
		return nil, err
	}

	err = p.Channel.ExchangeDeclare(
		p.Exchange.options.Name,
		p.Exchange.options.Type,
		p.Exchange.options.Durable,
		p.Exchange.options.AutoDelete,
		p.Exchange.options.Internal,
		p.Exchange.options.NoWait,
		p.Exchange.options.Arguments,
	)
	if err != nil {
		return nil, err
	}

	return p.Channel, nil
}

func (p *Producer) Publish(routingKey string, publishing amqp.Publishing, mandatory, immediate bool) error {
	ch, err := p.getChannel()
	if err != nil {
		p.CloseForErr(err)
		return fmt.Errorf("failed to get a channel: %s", err)
	}

	err = ch.Publish(p.Exchange.options.Name, routingKey, mandatory, immediate, publishing)
	if err != nil {
		p.CloseForErr(err)
	}
	return err
}

func (p *Producer) CloseForErr(oriErr error) {
	p.m.Lock()
	defer p.m.Unlock()

	if p.Channel != nil {
		_ = p.Channel.Close()
		p.Channel = nil
	}

	if oriErr == amqp.ErrClosed {
		_ = p.Exchange.refreshConn()
	}
}
