package log

import (
	"encoding/json"
	"regexp"
	"strings"
	"time"
)

type (
	_HTTPDataLogParser struct {
		regMatchLog *regexp.Regexp
	}
)

const (
	_HTTPLogMatchReg = "\\[HTTP\\]\t\\[.*\\]\tURL:.*"
)

// Parse 解析HTTP 调用日志
func (c _HTTPDataLogParser) Parse(logText string) (map[string]interface{}, bool) {
	params, matched := make(map[string]interface{}), c.regMatchLog.Match([]byte(logText))
	if matched {
		params["data_category"] = "HTTP"
		logArray := strings.Split(logText, "\t")
		arrayLen := len(logArray)
		if arrayLen > 1 {
			if t, err := time.ParseDuration(strings.Trim(strings.Trim(logArray[1], "["), "]")); err == nil {
				params["time_cost"] = t.Nanoseconds() //纳秒
			}
		}
		if arrayLen > 2 {
			params["url"] = strings.ReplaceAll(logArray[2], "URL:", "")
		}
		if arrayLen > 3 {
			params["request"] = logArray[3]
		}

		if arrayLen > 4 {
			rsp := strings.Trim(logArray[4], "Error:")
			if rsp == logArray[3] {
				rsp = strings.Trim(logArray[4], "Response:")
			}
			m := make(map[string]interface{})
			if err := json.Unmarshal([]byte(rsp), &m); err == nil {
				params["response"] = m
			} else {
				params["response"] = rsp
			}
		}
	}
	return params, matched
}
