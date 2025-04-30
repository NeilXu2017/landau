package log4go

import (
	"fmt"
	"os"
	"strings"
	"time"
)

type FileLogWriter struct {
	rec               chan *LogRecord //chan write
	filename          string          // The opened file
	file              *os.File
	format            string // The logging format
	header, trailer   string // File header/trailer
	maxlines          int    // Rotate at linecount
	maxlines_curlines int
	maxsize           int // Rotate at size
	maxsize_cursize   int
	daily             bool // Rotate daily
	daily_opendate    int
	rotate            bool // Keep old logfiles (.001, .002, etc)
	maxbackup         int
	sanitize          bool // Sanitize newlines to prevent log injection
	lastWriteLogTime  int64
}

func (w *FileLogWriter) LogWrite(rec *LogRecord) {
	w.rec <- rec
}

func (w *FileLogWriter) Close() {
	close(w.rec)
	w.file.Sync()
}

func NewFileLogWriter(fName string, rotate bool, daily bool) *FileLogWriter {
	w := &FileLogWriter{
		rec:       make(chan *LogRecord, LogBufferLength),
		filename:  fName,
		format:    "[%D %T] [%L] (%S) %M",
		daily:     daily,
		rotate:    rotate,
		maxbackup: 999,
		sanitize:  false,
	}
	checkFileLogger = append(checkFileLogger, _FileLogCheckStatus{w: w})
	w.intRotate(false)
	go func() {
		defer func() {
			if w.file != nil {
				fmt.Fprint(w.file, FormatLogRecord(w.trailer, &LogRecord{Created: time.Now()}))
				w.file.Close()
			}
			if e := recover(); e != nil {
				fmt.Printf("Panicing %s\n", e)
			}
		}()
		writeErrorCount, tryReOpenErrorsCount := 0, 10 //记录连续些日志出错累积错误次数, 尝试重新打开的条件次数 (累积连续错误次数)
		for {
			rec, ok := <-w.rec
			if !ok {
				return
			}
			now := time.Now()
			if (w.maxlines > 0 && w.maxlines_curlines >= w.maxlines) ||
				(w.maxsize > 0 && w.maxsize_cursize >= w.maxsize) ||
				(w.daily && now.Day() != w.daily_opendate) {
				w.intRotate(true)
			}
			if w.sanitize {
				rec.Message = strings.Replace(rec.Message, "\n", "\\n", -1)
			}
			if n, err := fmt.Fprint(w.file, FormatLogRecord(w.format, rec)); err == nil {
				w.maxlines_curlines++
				w.maxsize_cursize += n
				writeErrorCount = 0
			} else {
				writeErrorCount++
				if writeErrorCount%tryReOpenErrorsCount == 0 {
					w.intRotate(false)
				}
			}
			w.lastWriteLogTime = rec.Created.Unix()
		}
	}()
	return w
}

func (w *FileLogWriter) intRotate(rotateNow bool) {
	if w.file != nil {
		fmt.Fprint(w.file, FormatLogRecord(w.trailer, &LogRecord{Created: time.Now()}))
		w.file.Close()
	}
	if w.rotate {
		if info, err := os.Stat(w.filename); err == nil {
			modTime := info.ModTime()
			w.daily_opendate = modTime.Day()
			switch {
			case w.daily == false && rotateNow:
				for num := w.maxbackup - 1; num >= 1; num-- {
					fName := w.filename + fmt.Sprintf(".%d", num)
					nfName := w.filename + fmt.Sprintf(".%d", num+1)
					if _, err = os.Lstat(fName); err == nil {
						if err = os.Rename(fName, nfName); err != nil {
							fmt.Printf("Rotate os.Rename %s to %s error:%s\n ", fName, nfName, err.Error())
						}
					} else {
						fmt.Printf("Rotate os.Lstat %s error:%s\n ", fName, err.Error())
					}
				}
			case w.daily && time.Now().Day() != w.daily_opendate:
				modDate := modTime.Format("2006-01-02")
				fName := w.filename + fmt.Sprintf(".%s", modDate)
				w.file.Close()
				if err = os.Rename(w.filename, fName); err != nil {
					fmt.Printf("Rotate os.Rename %s to %s error:%s\n ", w.filename, fName, err.Error())
				}
			}
		}
	}
	fd, err := os.OpenFile(w.filename, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0660)
	if err != nil {
		fmt.Printf("os.OpenFile %s error:%s\n", w.filename, err.Error())
		return
	}
	w.file = fd
	now := time.Now()
	fmt.Fprint(w.file, FormatLogRecord(w.header, &LogRecord{Created: now}))
	w.daily_opendate = now.Day()
	w.maxlines_curlines = 0
	w.maxsize_cursize = 0
}

func (w *FileLogWriter) SetFormat(format string) *FileLogWriter {
	w.format = format
	return w
}

func (w *FileLogWriter) SetHeadFoot(head, foot string) *FileLogWriter {
	w.header, w.trailer = head, foot
	if w.maxlines_curlines == 0 {
		fmt.Fprint(w.file, FormatLogRecord(w.header, &LogRecord{Created: time.Now()}))
	}
	return w
}

func (w *FileLogWriter) SetRotateLines(maxlines int) *FileLogWriter {
	w.maxlines = maxlines
	return w
}

func (w *FileLogWriter) SetRotateSize(maxsize int) *FileLogWriter {
	w.maxsize = maxsize
	return w
}

func (w *FileLogWriter) SetRotateDaily(daily bool) *FileLogWriter {
	w.daily = daily
	return w
}

func (w *FileLogWriter) SetRotateMaxBackup(maxbackup int) *FileLogWriter {
	w.maxbackup = maxbackup
	return w
}

func (w *FileLogWriter) SetRotate(rotate bool) *FileLogWriter {
	w.rotate = rotate
	return w
}

func (w *FileLogWriter) SetSanitize(sanitize bool) *FileLogWriter {
	w.sanitize = sanitize
	return w
}

func NewXMLLogWriter(fName string, rotate bool, daily bool) *FileLogWriter {
	return NewFileLogWriter(fName, rotate, daily).
		SetFormat(`<record level="%L"><timestamp>%D %T</timestamp><source>%S</source><message>%M</message></record>`).
		SetHeadFoot(`<log created="%D %T">`, "</log>")
}
