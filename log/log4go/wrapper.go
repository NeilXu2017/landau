package log4go

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

var (
	Global Logger
)

func init() {
	Global = NewDefaultLogger(FINE)
}

func LoadConfiguration(filename string, types ...string) {
	if len(types) > 0 && types[0] == "xml" {
		Global.LoadConfiguration(filename)
	} else {
		Global.LoadJsonConfiguration(filename)
	}
}

func AddFilter(name string, lvl Level, writer LogWriter) {
	Global.AddFilter(name, lvl, writer)
}

func Close() {
	Global.Close()
	ticker.Stop()
}

func Crash(args ...interface{}) {
	if len(args) > 0 {
		Global.intLogf(CRITICAL, strings.Repeat(" %v", len(args))[1:], args...)
	}
	panic(args)
}
func Crashf(format string, args ...interface{}) {
	Global.intLogf(CRITICAL, format, args...)
	Global.Close() // so that hopefully the messages get logged
	panic(fmt.Sprintf(format, args...))
}

func Exit(args ...interface{}) {
	if len(args) > 0 {
		Global.intLogf(ERROR, strings.Repeat(" %v", len(args))[1:], args...)
	}
	Global.Close() // so that hopefully the messages get logged
	os.Exit(0)
}

func Exitf(format string, args ...interface{}) {
	Global.intLogf(ERROR, format, args...)
	Global.Close() // so that hopefully the messages get logged
	os.Exit(0)
}

func Stderr(args ...interface{}) {
	if len(args) > 0 {
		Global.intLogf(ERROR, strings.Repeat(" %v", len(args))[1:], args...)
	}
}

func Stderrf(format string, args ...interface{}) {
	Global.intLogf(ERROR, format, args...)
}

func Stdout(args ...interface{}) {
	if len(args) > 0 {
		Global.intLogf(INFO, strings.Repeat(" %v", len(args))[1:], args...)
	}
}

func Stdoutf(format string, args ...interface{}) {
	Global.intLogf(INFO, format, args...)
}

func Log(lvl Level, source, message string) {
	Global.Log(lvl, source, message)
}

func Logf(lvl Level, format string, args ...interface{}) {
	Global.intLogf(lvl, format, args...)
}

func Logc(lvl Level, closure func() string) {
	Global.intLogc(lvl, closure)
}

func Finest(arg0 interface{}, args ...interface{}) {
	switch first := arg0.(type) {
	case string:
		Global.intLogf(FINEST, first, args...)
	case func() string:
		Global.intLogc(FINEST, first)
	default:
		Global.intLogf(FINEST, fmt.Sprint(arg0)+strings.Repeat(" %v", len(args)), args...)
	}
}

func Fine(arg0 interface{}, args ...interface{}) {
	switch first := arg0.(type) {
	case string:
		Global.intLogf(FINE, first, args...)
	case func() string:
		Global.intLogc(FINE, first)
	default:
		Global.intLogf(FINE, fmt.Sprint(arg0)+strings.Repeat(" %v", len(args)), args...)
	}
}

func Debug(arg0 interface{}, args ...interface{}) {
	switch first := arg0.(type) {
	case string:
		Global.intLogf(DEBUG, first, args...)
	case func() string:
		Global.intLogc(DEBUG, first)
	default:
		Global.intLogf(DEBUG, fmt.Sprint(arg0)+strings.Repeat(" %v", len(args)), args...)
	}
}

func Trace(arg0 interface{}, args ...interface{}) {
	switch first := arg0.(type) {
	case string:
		Global.intLogf(TRACE, first, args...)
	case func() string:
		Global.intLogc(TRACE, first)
	default:
		Global.intLogf(TRACE, fmt.Sprint(arg0)+strings.Repeat(" %v", len(args)), args...)
	}
}

func Info(arg0 interface{}, args ...interface{}) {
	switch first := arg0.(type) {
	case string:
		Global.intLogf(INFO, first, args...)
	case func() string:
		Global.intLogc(INFO, first)
	default:
		Global.intLogf(INFO, fmt.Sprint(arg0)+strings.Repeat(" %v", len(args)), args...)
	}
}

func Warn(arg0 interface{}, args ...interface{}) error {
	switch first := arg0.(type) {
	case string:
		Global.intLogf(WARNING, first, args...)
		return errors.New(fmt.Sprintf(first, args...))
	case func() string:
		str := first()
		Global.intLogf(WARNING, "%s", str)
		return errors.New(str)
	default:
		Global.intLogf(WARNING, fmt.Sprint(first)+strings.Repeat(" %v", len(args)), args...)
		return errors.New(fmt.Sprint(first) + fmt.Sprintf(strings.Repeat(" %v", len(args)), args...))
	}
}

func Error(arg0 interface{}, args ...interface{}) error {
	switch first := arg0.(type) {
	case string:
		Global.intLogf(ERROR, first, args...)
		return errors.New(fmt.Sprintf(first, args...))
	case func() string:
		str := first()
		Global.intLogf(ERROR, "%s", str)
		return errors.New(str)
	default:
		Global.intLogf(ERROR, fmt.Sprint(first)+strings.Repeat(" %v", len(args)), args...)
		return errors.New(fmt.Sprint(first) + fmt.Sprintf(strings.Repeat(" %v", len(args)), args...))
	}
}

func Critical(arg0 interface{}, args ...interface{}) error {
	switch first := arg0.(type) {
	case string:
		Global.intLogf(CRITICAL, first, args...)
		return errors.New(fmt.Sprintf(first, args...))
	case func() string:
		str := first()
		Global.intLogf(CRITICAL, "%s", str)
		return errors.New(str)
	default:
		Global.intLogf(CRITICAL, fmt.Sprint(first)+strings.Repeat(" %v", len(args)), args...)
		return errors.New(fmt.Sprint(first) + fmt.Sprintf(strings.Repeat(" %v", len(args)), args...))
	}
}
