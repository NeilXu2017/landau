package log

import (
	"regexp"
	"strings"
	"time"
)

type (
	_GRPCDataLogParser struct {
		regMatchLog *regexp.Regexp
	}
)

const (
	_GRPCLogMatchReg = "\\[GRPC\\]\t\\[.*\\]\t\\[.*\\]\tRequest:.*"
)

// Parse 解析HTTP 调用日志
func (c _GRPCDataLogParser) Parse(logText string) (map[string]interface{}, bool) {
	params, matched := make(map[string]interface{}), c.regMatchLog.Match([]byte(logText))
	if matched {
		params["data_category"] = "GRPC"
		logArray := strings.Split(logText, "\t")
		arrayLen := len(logArray)
		if arrayLen > 1 {
			if t, err := time.ParseDuration(strings.Trim(strings.Trim(logArray[1], "["), "]")); err == nil {
				params["time_cost"] = t.Nanoseconds() //纳秒
			}
		}
		if arrayLen > 2 {
			params["service_method"] = strings.Trim(strings.Trim(logArray[2], "["), "]")
		}
		if arrayLen > 3 {
			params["parameter"] = strings.TrimLeft(logArray[3], "Request:")
		}
		if arrayLen > 4 {
			params["result"] = strings.TrimLeft(logArray[4], "Response:")
		}
		if arrayLen > 5 {
			if logArray[5] != "" {
				params["error"] = strings.TrimLeft(logArray[4], "Error:")
			}
		}
	}
	return params, matched
}
