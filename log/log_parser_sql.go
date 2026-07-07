package log

import (
	"regexp"
	"strings"
	"time"
)

type (
	_SQLDataLogParser struct {
		regMatchLog *regexp.Regexp
	}
)

const (
	_SQLLogMatchReg = "\\[SQL\\] \\[.*\\]\tArgs \\[.*"
)

// Parse 解析HTTP 调用日志
func (c _SQLDataLogParser) Parse(logText string) (map[string]interface{}, bool) {
	params, matched := make(map[string]interface{}), c.regMatchLog.Match([]byte(logText))
	if matched {
		params["data_category"] = "SQL"
		logArray := strings.Split(logText, "\t")
		arrayLen := len(logArray)
		if arrayLen > 0 {
			if t, err := time.ParseDuration(strings.Trim(strings.Trim(strings.ReplaceAll(logArray[0], "[SQL] ", ""), "["), "]")); err == nil {
				params["time_cost"] = t.Nanoseconds() //纳秒
			}
		}
		if arrayLen > 1 {
			params["SQL"] = strings.Trim(strings.Trim(logArray[1], "["), "]")
		}
		if arrayLen > 2 {
			params["parameter"] = strings.Trim(strings.Trim(strings.ReplaceAll(logArray[2], "Args ", ""), "["), "]")
		}
		if arrayLen > 3 {
			params["result"] = strings.Trim(strings.Trim(strings.Trim(logArray[3], "Error:"), "["), "]")
		}
	}
	return params, matched
}
