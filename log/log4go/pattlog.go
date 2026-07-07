package log4go

import (
	"bytes"
	"fmt"
	"io"
	"strings"
)

type FormatLogWriter chan *LogRecord

// FormatLogRecord  Known format codes:
// %Z - Time (15:04:05.999999999)
// %T - Time (15:04:05 MST)
// %t - Time (15:04)
// %D - Date (2006/01/02)
// %d - Date (01/02/06)
// %L - Level (FNST, FINE, DEBG, TRAC, WARN, EROR, CRIT)
// %S - Source
// %M - Message
// Ignores unknown formats
// Recommended: "[%D %T] [%L] (%S) %M"
func FormatLogRecord(format string, rec *LogRecord) string {
	if rec == nil {
		return "<nil>"
	}
	if len(format) == 0 {
		return ""
	}
	out := bytes.NewBuffer(make([]byte, 0, 64))
	year, month, day := rec.Created.Date()
	hour, minute, second, nanosecond := rec.Created.Hour(), rec.Created.Minute(), rec.Created.Second(), rec.Created.Nanosecond()
	zone, _ := rec.Created.Zone()
	pieces := bytes.Split([]byte(format), []byte{'%'})
	for i, piece := range pieces {
		if i > 0 && len(piece) > 0 {
			switch piece[0] {
			case 'Z':
				out.WriteString(fmt.Sprintf("%02d:%02d:%02d.%09d %s", hour, minute, second, nanosecond, zone))
			case 'T':
				out.WriteString(fmt.Sprintf("%02d:%02d:%02d %s", hour, minute, second, zone))
			case 't':
				out.WriteString(fmt.Sprintf("%02d:%02d", hour, minute))
			case 'D':
				out.WriteString(fmt.Sprintf("%04d/%02d/%02d", year, month, day))
			case 'd':
				out.WriteString(fmt.Sprintf("%02d/%02d/%02d", day, month, year%100))
			case 'L':
				out.WriteString(levelStrings[rec.Level])
			case 'S':
				out.WriteString(rec.Source)
			case 's':
				slice := strings.Split(rec.Source, "/")
				out.WriteString(slice[len(slice)-1])
			case 'M':
				out.WriteString(rec.Message)
			case 'C':
				if len(rec.Category) == 0 {
					rec.Category = "DEFAULT"
				}
				out.WriteString(rec.Category)
			}
			if len(piece) > 1 {
				out.Write(piece[1:])
			}
		} else if len(piece) > 0 {
			out.Write(piece)
		}
	}
	out.WriteByte('\n')
	return out.String()
}

func NewFormatLogWriter(out io.Writer, format string) FormatLogWriter {
	records := make(FormatLogWriter, LogBufferLength)
	go records.run(out, format)
	return records
}

func (w FormatLogWriter) run(out io.Writer, format string) {
	defer func() {
		if e := recover(); e != nil {
			fmt.Printf("Panicing %s\n", e)
		}
	}()
	for rec := range w {
		_, _ = fmt.Fprint(out, FormatLogRecord(format, rec))
	}
}

func (w FormatLogWriter) LogWrite(rec *LogRecord) {
	w <- rec
}

func (w FormatLogWriter) Close() {
	close(w)
}
