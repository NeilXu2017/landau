package uamqp

import (
	"fmt"
	"net"
	"strconv"
)

type NodeOptions struct {
	Host     string
	Port     int
	User     string
	Password string
}

func (n NodeOptions) GetURI() string {
	return fmt.Sprintf("amqp://%s:%s@%s/", n.User, n.Password, net.JoinHostPort(n.Host, strconv.Itoa(n.Port)))
}
