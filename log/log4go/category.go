package log4go

import (
	"fmt"
	"runtime"
	"strings"
	"time"
)

func LOGGER(category string) *Filter {
	f, ok := Global[category]
	if !ok {
		f = &Filter{CRITICAL, NewConsoleLogWriter(), "DEFAULT"}
	} else {
		f.Category = category
	}
	return f
}

func (f *Filter) intLogf(lvl Level, format string, args ...interface{}) {
	if lvl >= f.Level {
		pc, _, lineno, ok := runtime.Caller(3)
		src := ""
		if ok {
			src = fmt.Sprintf("%s:%d", runtime.FuncForPC(pc).Name(), lineno)
		}
		msg := format
		if len(args) > 0 {
			msg = fmt.Sprintf(format, args...)
		}
		rec := &LogRecord{Level: lvl, Created: time.Now(), Source: src, Message: msg, Category: f.Category}
		if f.Category != "DEFAULT" && f.Category != "stdout" {
			f.LogWrite(rec)
		}
		if defaultFilter := Global["stdout"]; defaultFilter != nil && lvl >= defaultFilter.Level {
			defaultFilter.LogWrite(rec)
		}
	}
}

func (f *Filter) intLogc(lvl Level, closure func() string) {
	if lvl >= f.Level {
		pc, _, lineno, ok := runtime.Caller(3)
		src := ""
		if ok {
			src = fmt.Sprintf("%s:%d", runtime.FuncForPC(pc).Name(), lineno)
		}
		rec := &LogRecord{Level: lvl, Created: time.Now(), Source: src, Message: closure(), Category: f.Category}
		if f.Category != "DEFAULT" && f.Category != "stdout" {
			f.LogWrite(rec)
		}
		if defaultFilter := Global["stdout"]; defaultFilter != nil && lvl > defaultFilter.Level {
			defaultFilter.LogWrite(rec)
		}
	}
}

func (f *Filter) Log(lvl Level, source, message string) {
	if lvl >= f.Level {
		rec := &LogRecord{Level: lvl, Created: time.Now(), Source: source, Message: message, Category: f.Category}
		if f.Category != "DEFAULT" && f.Category != "stdout" {
			f.LogWrite(rec)
		}
		if defaultFilter := Global["stdout"]; defaultFilter != nil && lvl > defaultFilter.Level {
			defaultFilter.LogWrite(rec)
		}
	}
}

func (f *Filter) Logf(lvl Level, format string, args ...interface{}) {
	f.intLogf(lvl, format, args...)
}

func (f *Filter) Logc(lvl Level, closure func() string) {
	f.intLogc(lvl, closure)
}

func (f *Filter) Finest(arg0 interface{}, args ...interface{}) {
	switch first := arg0.(type) {
	case string:
		f.intLogf(FINEST, first, args...)
	case func() string:
		f.intLogc(FINEST, first)
	default:
		f.intLogf(FINEST, fmt.Sprint(arg0)+strings.Repeat(" %v", len(args)), args...)
	}
}

func (f *Filter) Fine(arg0 interface{}, args ...interface{}) {
	switch first := arg0.(type) {
	case string:
		f.intLogf(FINE, first, args...)
	case func() string:
		f.intLogc(FINE, first)
	default:
		f.intLogf(FINE, fmt.Sprint(arg0)+strings.Repeat(" %v", len(args)), args...)
	}
}

func (f *Filter) Debug(arg0 interface{}, args ...interface{}) {
	switch first := arg0.(type) {
	case string:
		f.intLogf(DEBUG, first, args...)
	case func() string:
		f.intLogc(DEBUG, first)
	default:
		f.intLogf(DEBUG, fmt.Sprint(arg0)+strings.Repeat(" %v", len(args)), args...)
	}
}

func (f *Filter) Trace(arg0 interface{}, args ...interface{}) {
	switch first := arg0.(type) {
	case string:
		f.intLogf(TRACE, first, args...)
	case func() string:
		f.intLogc(TRACE, first)
	default:
		f.intLogf(TRACE, fmt.Sprint(arg0)+strings.Repeat(" %v", len(args)), args...)
	}
}

func (f *Filter) Info(arg0 interface{}, args ...interface{}) {
	switch first := arg0.(type) {
	case string:
		f.intLogf(INFO, first, args...)
	case func() string:
		f.intLogc(INFO, first)
	default:
		f.intLogf(INFO, fmt.Sprint(arg0)+strings.Repeat(" %v", len(args)), args...)
	}
}

func (f *Filter) Warn(arg0 interface{}, args ...interface{}) {
	switch first := arg0.(type) {
	case string:
		f.intLogf(WARNING, first, args...)
	case func() string:
		f.intLogc(WARNING, first)
	default:
		f.intLogf(WARNING, fmt.Sprint(arg0)+strings.Repeat(" %v", len(args)), args...)
	}
}

func (f *Filter) Error(arg0 interface{}, args ...interface{}) {
	switch first := arg0.(type) {
	case string:
		f.intLogf(ERROR, first, args...)
	case func() string:
		f.intLogc(ERROR, first)
	default:
		f.intLogf(ERROR, fmt.Sprint(arg0)+strings.Repeat(" %v", len(args)), args...)
	}
}

func (f *Filter) Critical(arg0 interface{}, args ...interface{}) {
	switch first := arg0.(type) {
	case string:
		f.intLogf(CRITICAL, first, args...)
	case func() string:
		f.intLogc(CRITICAL, first)
	default:
		f.intLogf(CRITICAL, fmt.Sprint(arg0)+strings.Repeat(" %v", len(args)), args...)
	}
}
