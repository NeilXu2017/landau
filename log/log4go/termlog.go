package log4go

import (
	"fmt"
	"io"
	"os"
	"time"
)

var stdout io.Writer = os.Stdout

type ConsoleLogWriter struct {
	format string
	w      chan *LogRecord
}

func NewConsoleLogWriter() *ConsoleLogWriter {
	consoleWriter := &ConsoleLogWriter{format: "[%T %D] [%C] [%L] (%S) %M", w: make(chan *LogRecord, LogBufferLength)}
	go consoleWriter.run(stdout)
	return consoleWriter
}
func (c *ConsoleLogWriter) SetFormat(format string) {
	c.format = format
}
func (c *ConsoleLogWriter) run(out io.Writer) {
	for rec := range c.w {
		_, _ = fmt.Fprint(out, FormatLogRecord(c.format, rec))
	}
}

func (c *ConsoleLogWriter) LogWrite(rec *LogRecord) {
	c.w <- rec
}
func (c *ConsoleLogWriter) Close() {
	close(c.w)
	time.Sleep(50 * time.Millisecond) // Try to give console I/O time to complete
}
