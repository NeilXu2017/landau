package log4go

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type (
	ConsoleConfig struct {
		Enable  bool   `json:"enable"`
		Level   string `json:"level"`
		Pattern string `json:"pattern"`
	}
	FileConfig struct {
		Enable   bool   `json:"enable"`
		Category string `json:"category"`
		Level    string `json:"level"`
		Filename string `json:"filename"`
		Pattern  string `json:"pattern"`
		Rotate   bool   `json:"rotate"`
		MaxSize  string `json:"maxsize"`  // \d+[KMG]? Suffixes are in terms of 2**10
		MaxLines string `json:"maxlines"` //\d+[KMG]? Suffixes are in terms of thousands
		Daily    bool   `json:"daily"`    //Automatically rotates by day
		Sanitize bool   `json:"sanitize"` //Sanitize newlines to prevent log injection
	}
	SocketConfig struct {
		Enable   bool   `json:"enable"`
		Category string `json:"category"`
		Level    string `json:"level"`
		Pattern  string `json:"pattern"`
		Addr     string `json:"addr"`
		Protocol string `json:"protocol"`
	}
	LogConfig struct {
		Console *ConsoleConfig  `json:"console"`
		Files   []*FileConfig   `json:"files"`
		Sockets []*SocketConfig `json:"sockets"`
	}
)

func (log Logger) LoadJsonConfiguration(filename string) {
	log.Close()
	dst := new(bytes.Buffer)
	var (
		lc      LogConfig
		content string
	)
	err := json.Compact(dst, []byte(filename))
	if err != nil {
		content, err = ReadFile(filename)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "LoadJsonConfiguration: Error: Could not read %q: %s\n", filename, err)
			os.Exit(1)
		}
	} else {
		content = string(dst.Bytes())
	}
	err = json.Unmarshal([]byte(content), &lc)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "LoadJsonConfiguration: Error: Could not parse json configuration in %q: %s\n", filename, err)
		os.Exit(1)
	}
	if lc.Console.Enable {
		log["stdout"] = &Filter{getLogLevel(lc.Console.Level), newConsoleLogWriter(lc.Console), "DEFAULT"}
	}
	for _, fc := range lc.Files {
		if fc.Enable {
			if len(fc.Category) == 0 {
				_, _ = fmt.Fprintf(os.Stderr, "LoadJsonConfiguration: file category can not be empty in <%s>: ", filename)
				os.Exit(1)
			}
			log[fc.Category] = &Filter{getLogLevel(fc.Level), newFileLogWriter(fc), fc.Category}
		}
	}
	for _, sc := range lc.Sockets {
		if sc.Enable {
			if len(sc.Category) == 0 {
				_, _ = fmt.Fprintf(os.Stderr, "LoadJsonConfiguration: file category can not be empty in <%s>: ", filename)
				os.Exit(1)
			}
			log[sc.Category] = &Filter{getLogLevel(sc.Level), newSocketLogWriter(sc), sc.Category}
		}
	}
}

func getLogLevel(l string) Level {
	var lvl Level
	switch l {
	case "FINEST":
		lvl = FINEST
	case "FINE":
		lvl = FINE
	case "DEBUG":
		lvl = DEBUG
	case "TRACE":
		lvl = TRACE
	case "INFO":
		lvl = INFO
	case "WARNING":
		lvl = WARNING
	case "ERROR":
		lvl = ERROR
	case "CRITICAL":
		lvl = CRITICAL
	default:
		_, _ = fmt.Fprintf(os.Stderr, "LoadJsonConfiguration: Error: Required level <%s> for filter has unknown value: %s\n", "level", l)
		os.Exit(1)
	}
	return lvl
}

func newConsoleLogWriter(cf *ConsoleConfig) *ConsoleLogWriter {
	if cf.Enable {
		format := "[%D %T] [%C] [%L] (%S) %M"
		if len(cf.Pattern) > 0 {
			format = strings.Trim(cf.Pattern, " \r\n")
		}
		clw := NewConsoleLogWriter()
		clw.SetFormat(format)
		return clw
	}
	return nil
}

func newFileLogWriter(ff *FileConfig) *FileLogWriter {
	if ff.Enable {
		file, format := "app.log", "[%D %T] [%C] [%L] (%S) %M"
		maxLines, maxSize, daily, rotate, sanitize := 0, 0, false, false, false
		if len(ff.Filename) > 0 {
			file = ff.Filename
		}
		if len(ff.Pattern) > 0 {
			format = strings.Trim(ff.Pattern, " \r\n")
		}
		if len(ff.MaxLines) > 0 {
			maxLines = strToNumSuffix(strings.Trim(ff.MaxLines, " \r\n"), 1000)
		}
		if len(ff.MaxSize) > 0 {
			maxSize = strToNumSuffix(strings.Trim(ff.MaxSize, " \r\n"), 1024)
		}
		daily = ff.Daily
		rotate = ff.Rotate
		sanitize = ff.Sanitize
		flw := NewFileLogWriter(file, rotate, daily)
		flw.SetFormat(format)
		flw.SetRotateLines(maxLines)
		flw.SetRotateSize(maxSize)
		flw.SetSanitize(sanitize)
		return flw
	}
	return nil
}

func newSocketLogWriter(sf *SocketConfig) *SocketLogWriter {
	if sf.Enable && sf.Addr != "" && (sf.Protocol == "tcp" || sf.Protocol == "udp") {
		return NewSocketLogWriter(sf.Protocol, sf.Addr)
	}
	return nil
}

func ReadFile(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("[%s] path empty", path)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read file %s fail %s", path, err)
	}
	return strings.TrimSpace(string(b)), nil
}
