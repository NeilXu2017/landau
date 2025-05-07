package log4go

import (
	"errors"
	"fmt"
	"runtime"
	"strings"
	"time"
)

const (
	FINEST Level = iota
	FINE
	DEBUG
	TRACE
	INFO
	WARNING
	ERROR
	CRITICAL
)

type (
	LogRecord struct {
		Level    Level     // The log level
		Created  time.Time // The time at which the log message was created (nanoseconds)
		Source   string    // The message source
		Message  string    // The log message
		Category string    // The log group
	}
	LogWriter interface {
		LogWrite(rec *LogRecord)
		Close()
	}
	Filter struct {
		Level Level
		LogWriter
		Category string
	}
	Level  int
	Logger map[string]*Filter
)

var (
	levelStrings    = [...]string{"FNST", "FINE", "DEBG", "TRAC", "INFO", "WARN", "EROR", "CRIT"}
	LogBufferLength = 32
)

func (l Level) String() string {
	if l < 0 || int(l) > len(levelStrings) {
		return "UNKNOWN"
	}
	return levelStrings[int(l)]
}

func NewDefaultLogger(lvl Level) Logger {
	return Logger{"stdout": &Filter{lvl, NewConsoleLogWriter(), "DEFAULT"}}
}

func (log Logger) Close() {
	for name, f := range log {
		f.Close()
		delete(log, name)
	}
}

func (log Logger) AddFilter(name string, lvl Level, writer LogWriter, category ...string) Logger {
	c := "DEFAULT"
	if len(category) > 0 {
		c = category[0]
	}
	log[name] = &Filter{lvl, writer, c}
	return log
}

func (log Logger) Log(lvl Level, source, message string) {
	rec := &LogRecord{Level: lvl, Created: time.Now(), Source: source, Message: message}
	for _, filter := range log {
		if lvl >= filter.Level {
			filter.LogWrite(rec)
		}
	}
}

func (log Logger) intLogf(lvl Level, format string, args ...interface{}) {
	src := ""
	if pc, _, lineno, ok := runtime.Caller(3); ok {
		src = fmt.Sprintf("%s:%d", runtime.FuncForPC(pc).Name(), lineno)
	}
	msg := format
	if len(args) > 0 {
		msg = fmt.Sprintf(format, args...)
	}
	log.Log(lvl, src, msg)
}

func (log Logger) intLogc(lvl Level, closure func() string) {
	src := ""
	if pc, _, lineno, ok := runtime.Caller(3); ok {
		src = fmt.Sprintf("%s:%d", runtime.FuncForPC(pc).Name(), lineno)
	}
	log.Log(lvl, src, closure())
}

func (log Logger) Logf(lvl Level, format string, args ...interface{}) {
	log.intLogf(lvl, format, args...)
}

func (log Logger) Logc(lvl Level, closure func() string) {
	log.intLogc(lvl, closure)
}

func (log Logger) Finest(arg0 interface{}, args ...interface{}) {
	switch first := arg0.(type) {
	case string:
		log.intLogf(FINEST, first, args...)
	case func() string:
		log.intLogc(FINEST, first)
	default:
		log.intLogf(FINEST, fmt.Sprint(arg0)+strings.Repeat(" %v", len(args)), args...)
	}
}

func (log Logger) Fine(arg0 interface{}, args ...interface{}) {
	switch first := arg0.(type) {
	case string:
		log.intLogf(FINE, first, args...)
	case func() string:
		log.intLogc(FINE, first)
	default:
		log.intLogf(FINE, fmt.Sprint(arg0)+strings.Repeat(" %v", len(args)), args...)
	}
}

func (log Logger) Debug(arg0 interface{}, args ...interface{}) {
	switch first := arg0.(type) {
	case string:
		log.intLogf(DEBUG, first, args...)
	case func() string:
		log.intLogc(DEBUG, first)
	default:
		log.intLogf(DEBUG, fmt.Sprint(arg0)+strings.Repeat(" %v", len(args)), args...)
	}
}

func (log Logger) Trace(arg0 interface{}, args ...interface{}) {
	switch first := arg0.(type) {
	case string:
		log.intLogf(TRACE, first, args...)
	case func() string:
		log.intLogc(TRACE, first)
	default:
		log.intLogf(TRACE, fmt.Sprint(arg0)+strings.Repeat(" %v", len(args)), args...)
	}
}

func (log Logger) Info(arg0 interface{}, args ...interface{}) {
	switch first := arg0.(type) {
	case string:
		log.intLogf(INFO, first, args...)
	case func() string:
		log.intLogc(INFO, first)
	default:
		log.intLogf(INFO, fmt.Sprint(arg0)+strings.Repeat(" %v", len(args)), args...)
	}
}

func (log Logger) Warn(arg0 interface{}, args ...interface{}) error {
	var msg string
	switch first := arg0.(type) {
	case string:
		msg = fmt.Sprintf(first, args...)
	case func() string:
		msg = first()
	default:
		msg = fmt.Sprintf(fmt.Sprint(first)+strings.Repeat(" %v", len(args)), args...)
	}
	log.intLogf(WARNING, msg)
	return errors.New(msg)
}

func (log Logger) Error(arg0 interface{}, args ...interface{}) error {
	var msg string
	switch first := arg0.(type) {
	case string:
		msg = fmt.Sprintf(first, args...)
	case func() string:
		msg = first()
	default:
		msg = fmt.Sprintf(fmt.Sprint(first)+strings.Repeat(" %v", len(args)), args...)
	}
	log.intLogf(ERROR, msg)
	return errors.New(msg)
}

func (log Logger) Critical(arg0 interface{}, args ...interface{}) error {
	var msg string
	switch first := arg0.(type) {
	case string:
		msg = fmt.Sprintf(first, args...)
	case func() string:
		msg = first()
	default:
		msg = fmt.Sprintf(fmt.Sprint(first)+strings.Repeat(" %v", len(args)), args...)
	}
	log.intLogf(CRITICAL, msg)
	return errors.New(msg)
}
