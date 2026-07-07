package log4go

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
)

type SocketLogWriter chan *LogRecord

func (w SocketLogWriter) LogWrite(rec *LogRecord) {
	w <- rec
}

func (w SocketLogWriter) Close() {
	close(w)
}

// NewSocketLogWriter current proto must be udp. tcp not resume connection
func NewSocketLogWriter(proto, addr string) *SocketLogWriter {
	sock, err := net.Dial(proto, addr)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "NewSocketLogWriter(%q): %s\n", addr, err)
		return nil
	}
	w := SocketLogWriter(make(chan *LogRecord, LogBufferLength))
	go func() {
		defer func() {
			if sock != nil && proto == "tcp" {
				_ = sock.Close()
			}
		}()
		for rec := range w {
			if b, err := json.Marshal(rec); err == nil && len(b) > 0 {
				if _, err = sock.Write(b); err != nil {
					_, _ = fmt.Fprintf(os.Stderr, "SocketLogWriter(%s): %s", addr, err.Error())
				}
			}
		}
	}()
	return &w
}
