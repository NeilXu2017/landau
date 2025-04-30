package log4go

import (
	"encoding/xml"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type (
	xmlProperty struct {
		Name  string `xml:"name,attr"`
		Value string `xml:",chardata"`
	}
	xmlFilter struct {
		Enabled  string        `xml:"enabled,attr"`
		Tag      string        `xml:"tag"`
		Level    string        `xml:"level"`
		Type     string        `xml:"type"`
		Property []xmlProperty `xml:"property"`
	}
	xmlLoggerConfig struct {
		Filter []xmlFilter `xml:"filter"`
	}
)

func (log Logger) LoadConfiguration(filename string) {
	log.Close()
	contents, err := os.ReadFile(filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "LoadConfiguration: Error: Could not ReadFile %q error:%s\n", filename, err.Error())
		os.Exit(1)
	}
	xc := new(xmlLoggerConfig)
	if err = xml.Unmarshal(contents, xc); err != nil {
		fmt.Fprintf(os.Stderr, "LoadConfiguration: Error: Could not parse XML configuration in %q: %s\n", filename, err)
		os.Exit(1)
	}
	for _, xmlfilt := range xc.Filter {
		var filt LogWriter
		var lvl Level
		bad, good, enabled := false, true, false
		if len(xmlfilt.Enabled) == 0 {
			fmt.Fprintf(os.Stderr, "LoadConfiguration: Error: Required attribute %s for filter missing in %s\n", "enabled", filename)
			bad = true
		} else {
			enabled = xmlfilt.Enabled != "false"
		}
		if len(xmlfilt.Tag) == 0 {
			fmt.Fprintf(os.Stderr, "LoadConfiguration: Error: Required child <%s> for filter missing in %s\n", "tag", filename)
			bad = true
		}
		if len(xmlfilt.Type) == 0 {
			fmt.Fprintf(os.Stderr, "LoadConfiguration: Error: Required child <%s> for filter missing in %s\n", "type", filename)
			bad = true
		}
		if len(xmlfilt.Level) == 0 {
			fmt.Fprintf(os.Stderr, "LoadConfiguration: Error: Required child <%s> for filter missing in %s\n", "level", filename)
			bad = true
		}

		switch xmlfilt.Level {
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
			fmt.Fprintf(os.Stderr, "LoadConfiguration: Error: Required child <%s> for filter has unknown value in %s: %s\n", "level", filename, xmlfilt.Level)
			bad = true
		}
		if bad {
			os.Exit(1)
		}
		if !enabled {
			continue
		}
		switch xmlfilt.Type {
		case "console":
			filt, good = xmlToConsoleLogWriter(filename, xmlfilt.Property)
		case "file":
			filt, good = xmlToFileLogWriter(filename, xmlfilt.Property)
		case "xml":
			filt, good = xmlToXMLLogWriter(filename, xmlfilt.Property)
		case "socket":
			filt, good = xmlToSocketLogWriter(filename, xmlfilt.Property)
		default:
			fmt.Fprintf(os.Stderr, "LoadConfiguration: Error: Could not load XML configuration in %s: unknown filter type \"%s\"\n", filename, xmlfilt.Type)
			os.Exit(1)
		}
		if !good {
			os.Exit(1)
		}
		log[xmlfilt.Tag] = &Filter{lvl, filt, "DEFAULT"}
	}
}

func xmlToConsoleLogWriter(filename string, props []xmlProperty) (*ConsoleLogWriter, bool) {
	format := "[%D %T] [%L] (%S) %M"
	for _, prop := range props {
		if prop.Name == "format" {
			format = strings.Trim(prop.Value, " \r\n")
		}
	}
	clw := NewConsoleLogWriter()
	clw.SetFormat(format)
	return clw, true
}

func strToNumSuffix(str string, mult int) int {
	num := 1
	if len(str) > 1 {
		switch str[len(str)-1] {
		case 'G', 'g':
			num *= mult
			fallthrough
		case 'M', 'm':
			num *= mult
			fallthrough
		case 'K', 'k':
			num *= mult
			str = str[0 : len(str)-1]
		}
	}
	parsed, _ := strconv.Atoi(str)
	return parsed * num
}

func xmlToFileLogWriter(filename string, props []xmlProperty) (*FileLogWriter, bool) {
	file, format := "", "[%D %T] [%L] (%S) %M"
	maxlines, maxsize, daily, rotate, sanitize := 0, 0, false, false, false
	for _, prop := range props {
		switch prop.Name {
		case "filename":
			file = strings.Trim(prop.Value, " \r\n")
		case "format":
			format = strings.Trim(prop.Value, " \r\n")
		case "maxlines":
			maxlines = strToNumSuffix(strings.Trim(prop.Value, " \r\n"), 1000)
		case "maxsize":
			maxsize = strToNumSuffix(strings.Trim(prop.Value, " \r\n"), 1024)
		case "daily":
			daily = strings.Trim(prop.Value, " \r\n") != "false"
		case "rotate":
			rotate = strings.Trim(prop.Value, " \r\n") != "false"
		case "sanitize":
			sanitize = strings.Trim(prop.Value, " \r\n") != "false"
		}
	}
	if file == "" {
		fmt.Fprintf(os.Stderr, "LoadConfiguration: Error: Required property \"%s\" for file filter missing in %s\n", "filename", filename)
		return nil, false
	}
	flw := NewFileLogWriter(file, rotate, daily)
	flw.SetFormat(format)
	flw.SetRotateLines(maxlines)
	flw.SetRotateSize(maxsize)
	flw.SetSanitize(sanitize)
	return flw, true
}

func xmlToXMLLogWriter(filename string, props []xmlProperty) (*FileLogWriter, bool) {
	file, maxrecords, maxsize, daily, rotate := "", 0, 0, false, false
	for _, prop := range props {
		switch prop.Name {
		case "filename":
			file = strings.Trim(prop.Value, " \r\n")
		case "maxrecords":
			maxrecords = strToNumSuffix(strings.Trim(prop.Value, " \r\n"), 1000)
		case "maxsize":
			maxsize = strToNumSuffix(strings.Trim(prop.Value, " \r\n"), 1024)
		case "daily":
			daily = strings.Trim(prop.Value, " \r\n") != "false"
		case "rotate":
			rotate = strings.Trim(prop.Value, " \r\n") != "false"
		}
	}
	if file == "" {
		fmt.Fprintf(os.Stderr, "LoadConfiguration: Error: Required property \"%s\" for xml filter missing in %s\n", "filename", filename)
		return nil, false
	}
	xlw := NewXMLLogWriter(file, rotate, daily)
	xlw.SetRotateLines(maxrecords)
	xlw.SetRotateSize(maxsize)
	return xlw, true
}

func xmlToSocketLogWriter(filename string, props []xmlProperty) (*SocketLogWriter, bool) {
	endpoint, protocol := "", "udp"
	for _, prop := range props {
		switch prop.Name {
		case "endpoint":
			endpoint = strings.Trim(prop.Value, " \r\n")
		case "protocol":
			protocol = strings.Trim(prop.Value, " \r\n")
		}
	}
	if endpoint == "" {
		fmt.Fprintf(os.Stderr, "LoadConfiguration: Error: Required property \"%s\" for file filter missing in %s\n", "endpoint", filename)
		return nil, false
	}
	return NewSocketLogWriter(protocol, endpoint), true
}
