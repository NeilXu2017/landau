package log

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jeanphorn/log4go"

	"github.com/NeilXu2017/landau/helper"
)

var (
	defaultLogger    = ""              //缺省logger 名称
	instanceID       string            //所在机器标识
	log2StdoutAsJSON map[string]string //日志JSON格式,重定向输出到标准输出 key 是  logger 名字, value 是 logger 的当前日志级别
	APILoggerName    = "API"           // APILoggerName api 日志名字
	GINLoggerName    = "gin"           // GINLoggerName gin 日志名字
	_stdout2File     = ""              //标准输出重定向到文件logger 名称
	_datalogParses   []DataLogParser
	_DoubleWrite     bool //日志是否双写模式
	_logLevelMap     = map[string]int{
		"DEBUG": 0,
		"INFO":  1,
		"WARN":  2,
		"ERROR": 3,
	}
	_ApplicationLoggerName       = make(map[string]string, 0) //应用配置的logger
	_ReplaceLoggerNameIfNotExist = "__replaced_logger_name__"
)

type (
	_loggerConfig struct {
		Category string `json:"category"` //名称
		Level    string `json:"level"`    //日志级别
		Stdout   bool   `json:"stdout"`   //是否JSON格式,重定向输出到标准输出
		FileName string `json:"filename"` //日志文件路径
	}
	_logConfig struct {
		Files             []_loggerConfig       `json:"files"`
		Stdout2File       string                `json:"stdout2File"`         //JSON 格式化日志重新输出到logger
		DoubleWriteMode   bool                  `json:"double_write"`        //日志是否双写模式
		FileMonitorConfig _LogFileMonitorConfig `json:"file_monitor_config"` //日志文件监控清理配置
	}
	//ConsoleLogger 替代标准输出
	ConsoleLogger struct {
		logger string
	}
	// DataLogParser 特定日志解析
	DataLogParser interface {
		Parse(logText string) (map[string]interface{}, bool) //解析参数,是否匹配
	}
)

func init() {
	instanceID, _ = os.Hostname()
	httpMatcher, _ := regexp.Compile(_HTTPLogMatchReg)
	p := _HTTPDataLogParser{regMatchLog: httpMatcher}
	RegisterDataLogParser(p)
	sqlMatcher, _ := regexp.Compile(_SQLLogMatchReg)
	sqlP := _SQLDataLogParser{regMatchLog: sqlMatcher}
	RegisterDataLogParser(sqlP)
	grpcMather, _ := regexp.Compile(_GRPCLogMatchReg)
	grpcP := _GRPCDataLogParser{regMatchLog: grpcMather}
	RegisterDataLogParser(grpcP)
}

// RegisterDataLogParser 注册日志解析
func RegisterDataLogParser(p DataLogParser) {
	_datalogParses = append(_datalogParses, p)
}

// NewConsoleLogger 返回标准输出替代实现类
func NewConsoleLogger(loggerName string) io.Writer {
	return &ConsoleLogger{logger: loggerName}
}

// Write 实现标准输出的方法,信息记录到指定的 logger 中，记录级别: Info
func (c *ConsoleLogger) Write(p []byte) (n int, err error) {
	msg := string(p)
	Info2(c.logger, msg)
	return len(p), nil
}

// LoadLogConfig 加载 Log 配置
// configContent 配置内容(JSON格式)或者是含有配置内容的文件名
// defaultLoggerName 缺省的日志Logger
func LoadLogConfig(configContent string, defaultLoggerName string) {
	defaultLogger = defaultLoggerName
	configContent = helper.ConvertAbsolutePath(configContent)
	log4go.LoadConfiguration(configContent)
	initLog2StdoutAsJSON(configContent)
}

func initLog2StdoutAsJSON(filename string) {
	dst := new(bytes.Buffer)
	lc, content := _logConfig{}, ""
	if err := json.Compact(dst, []byte(filename)); err == nil {
		content = dst.String()
	} else {
		content, err = log4go.ReadFile(filename)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "LoadJsonConfiguration: Error: Could not read %q: %s\n", filename, err)
			os.Exit(1)
		}
	}
	if err := json.Unmarshal([]byte(content), &lc); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "LoadJsonConfiguration: Error: Could not parse json configuration in %q: %s\n", filename, err)
		os.Exit(1)
	}
	_stdout2File = lc.Stdout2File
	_DoubleWrite = lc.DoubleWriteMode
	log2StdoutAsJSON = make(map[string]string)
	var logFiles []string
	for _, fc := range lc.Files {
		if fc.Stdout && fc.Category != "" {
			log2StdoutAsJSON[fc.Category] = fc.Level
		}
		logFiles = append(logFiles, fc.FileName)
		_ApplicationLoggerName[fc.Category] = ""
		if _, ok := _ApplicationLoggerName[_ReplaceLoggerNameIfNotExist]; !ok {
			if fc.Category != APILoggerName && fc.Category != GINLoggerName {
				_ApplicationLoggerName[_ReplaceLoggerNameIfNotExist] = fc.Category
			}
		}
	}
	if _, ok := _ApplicationLoggerName[_ReplaceLoggerNameIfNotExist]; !ok {
		if len(lc.Files) > 0 {
			_ApplicationLoggerName[_ReplaceLoggerNameIfNotExist] = lc.Files[0].Category
		}
	}
	_initMonitorConfig(lc.FileMonitorConfig, logFiles)
}

// Close 日志组件安全关闭
func Close() {
	log4go.Close()
}

// API logger
func _apiLoggerParse(logText string) map[string]interface{} {
	params := make(map[string]interface{})
	logArray := strings.Split(logText, "\t")
	arrayLen := len(logArray)
	if arrayLen > 0 {
		params["url"] = strings.Trim(strings.Trim(logArray[0], "["), "]")
	}
	if arrayLen > 1 {
		if t, err := time.ParseDuration(strings.Trim(strings.Trim(logArray[1], "["), "]")); err == nil {
			params["time_cost"] = t.Nanoseconds() //纳秒
		}
	}
	if arrayLen > 2 {
		params["custom_tag"] = strings.Trim(strings.Trim(logArray[2], "["), "]")
	}

	if arrayLen > 3 {
		requestParam := strings.Trim(logArray[3], "Request:")
		reqParamMap := make(map[string]interface{})
		if err := json.Unmarshal([]byte(requestParam), &reqParamMap); err == nil {
			params["request"] = reqParamMap
			if v, ok := reqParamMap["request_uuid"]; ok {
				params["request_uuid"] = v.(string)
			}
			if v, ok := reqParamMap["Action"]; ok {
				params["Action"] = v.(string)
			}
		} else {
			if urls, err := url.ParseQuery(requestParam); err == nil {
				urlParamMap := make(map[string]interface{})
				for k, v := range urls {
					if len(v) == 1 {
						urlParamMap[k] = v[0]
					} else {
						urlParamMap[k] = v
					}
					switch k {
					case "request_uuid", "Action":
						params[k] = v[0]
					}
				}
				params["request"] = urlParamMap

			} else {
				params["request"] = requestParam
			}
		}
	}
	if arrayLen > 4 {
		responseParam := strings.Trim(logArray[4], "Response:")
		responseMap := make(map[string]interface{})
		if err := json.Unmarshal([]byte(responseParam), &responseMap); err == nil {
			params["response"] = responseMap
		} else {
			params["response"] = responseParam
		}
	}
	return params
}

// gin logger
func _ginLoggerParse(logText string) map[string]interface{} {
	params := make(map[string]interface{})
	logText = strings.Trim(logText, "\n")
	logArray := strings.Split(logText, "|")
	arrayLen := len(logArray)
	if arrayLen == 1 {
		params["trace"] = logText
		return params
	}
	if arrayLen > 0 {
		params["request_time"] = strings.Trim(strings.ReplaceAll(logArray[0], "[GIN]", ""), " ")
	}
	if arrayLen > 1 {
		httpCode := strings.Trim(logArray[1], " ")
		if v, err := strconv.Atoi(httpCode); err == nil {
			params["response_http_code"] = v
		} else {
			params["response_http_code"] = httpCode
		}
	}
	if arrayLen > 2 {
		if t, err := time.ParseDuration(strings.Trim(logArray[2], " ")); err == nil {
			params["time_cost"] = t.Nanoseconds() //纳秒
		}
	}
	if arrayLen > 3 {
		params["client_ip"] = strings.Trim(logArray[3], " ")
	}
	if arrayLen > 4 {
		urlInfo := strings.Split(strings.Trim(logArray[4], " "), " ")
		if len(urlInfo) > 0 {
			params["request_method"] = urlInfo[0]
		}
		reqUrl, other := "", ""
		for i, j := 1, len(urlInfo); i < j; i++ {
			if urlInfo[i] != "" {
				if reqUrl == "" {
					reqUrl = urlInfo[i]
				} else {
					other = urlInfo[i]
				}
			}
		}
		params["url"] = reqUrl
		if other != "" {
			params["trace"] = other
		}
	}
	return params
}

func _mainLoggerParse(logText string) map[string]interface{} {
	for _, p := range _datalogParses {
		if m, ok := p.Parse(logText); ok {
			return m
		}
	}
	params := make(map[string]interface{})
	params["trace"] = logText
	return params
}

func _loggedText2Map(logger string, arg0 interface{}, args ...interface{}) map[string]interface{} {
	strArg0 := ""
	switch argV := arg0.(type) {
	case string:
		strArg0 = argV
	default:
		strArg0 = fmt.Sprintf("%s", arg0)
	}
	logText := fmt.Sprintf(strArg0, args...)
	switch logger {
	case APILoggerName:
		return _apiLoggerParse(logText)
	case GINLoggerName:
		return _ginLoggerParse(logText)
	default:
		return _mainLoggerParse(logText)
	}
}

// _log2StdoutAsJSON 返回是否处理了日志，如果处理了，原始的 logger 不需要记录
func _log2StdoutAsJSON(logger, level string, arg0 interface{}, args ...interface{}) bool {
	if logger == _stdout2File {
		return false
	}
	loggerLevel, ok := log2StdoutAsJSON[logger]
	if ok {
		if _logLevelMap[level] >= _logLevelMap[loggerLevel] { //需要记录日志
			logObject := map[string]interface{}{
				"logger":      logger,
				"level":       level,
				"instance_id": instanceID,
				"time":        time.Now().Unix(),
			}
			for k, v := range _loggedText2Map(logger, arg0, args...) {
				logObject[k] = v
			}
			if b, err := json.Marshal(logObject); err == nil {
				if _stdout2File == "" {
					fmt.Println(string(b))
				} else {
					switch level {
					case "INFO":
						log4go.LOGGER(_stdout2File).Info(string(b))
					case "DEBUG":
						log4go.LOGGER(_stdout2File).Debug(string(b))
					case "ERROR":
						log4go.LOGGER(_stdout2File).Error(string(b))
					case "WARN":
						log4go.LOGGER(_stdout2File).Warn(string(b))
					default:
						log4go.LOGGER(_stdout2File).Info(string(b))
					}
				}
			}
		}
	}
	return ok
}

func getLoggerSafely(logCategory string) string {
	if _, ok := _ApplicationLoggerName[logCategory]; ok {
		return logCategory
	}
	if v, ok := _ApplicationLoggerName[_ReplaceLoggerNameIfNotExist]; ok {
		return v
	}
	return logCategory
}

// Info 记录到缺省 Logger，级别 Info
func Info(arg0 interface{}, args ...interface{}) {
	if !_log2StdoutAsJSON(defaultLogger, "INFO", arg0, args...) || _DoubleWrite {
		log4go.LOGGER(getLoggerSafely(defaultLogger)).Info(arg0, args...)
	}
}

// Debug 记录到缺省 Logger，级别 Debug
func Debug(arg0 interface{}, args ...interface{}) {
	if !_log2StdoutAsJSON(defaultLogger, "DEBUG", arg0, args...) || _DoubleWrite {
		log4go.LOGGER(getLoggerSafely(defaultLogger)).Debug(arg0, args...)
	}
}

// Error 记录到缺省 Logger，级别 Error
func Error(arg0 interface{}, args ...interface{}) {
	if !_log2StdoutAsJSON(defaultLogger, "ERROR", arg0, args...) || _DoubleWrite {
		log4go.LOGGER(getLoggerSafely(defaultLogger)).Error(arg0, args...)
	}
}

// Warn 记录到缺省 Logger，级别 Warn
func Warn(arg0 interface{}, args ...interface{}) {
	if !_log2StdoutAsJSON(defaultLogger, "WARN", arg0, args...) || _DoubleWrite {
		log4go.LOGGER(getLoggerSafely(defaultLogger)).Warn(arg0, args...)
	}
}

// Info2 记录到指定的 logger，级别 Info
func Info2(logger string, arg0 interface{}, args ...interface{}) {
	if !_log2StdoutAsJSON(logger, "INFO", arg0, args...) || _DoubleWrite {
		log4go.LOGGER(getLoggerSafely(logger)).Info(arg0, args...)
	}
}

// Debug2 记录到指定的 logger，级别 Debug
func Debug2(logger string, arg0 interface{}, args ...interface{}) {
	if !_log2StdoutAsJSON(logger, "DEBUG", arg0, args...) || _DoubleWrite {
		log4go.LOGGER(getLoggerSafely(logger)).Debug(arg0, args...)
	}
}

// Error2 记录到指定的 logger，级别 Error
func Error2(logger string, arg0 interface{}, args ...interface{}) {
	if !_log2StdoutAsJSON(logger, "ERROR", arg0, args...) || _DoubleWrite {
		log4go.LOGGER(getLoggerSafely(logger)).Error(arg0, args...)
	}
}

// Warn2 记录到指定的 logger，级别 Warn
func Warn2(logger string, arg0 interface{}, args ...interface{}) {
	if !_log2StdoutAsJSON(logger, "WARN", arg0, args...) || _DoubleWrite {
		log4go.LOGGER(getLoggerSafely(logger)).Warn(arg0, args...)
	}
}
