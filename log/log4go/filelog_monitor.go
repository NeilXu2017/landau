package log4go

import (
	"os"
	"time"
)

type (
	_FileLogCheckStatus struct {
		w                   *FileLogWriter
		lastCheckTime       int64
		lastCheckFileLength int64
	}
)

var (
	checkFileLogger []_FileLogCheckStatus
	ticker          = time.NewTicker(30 * time.Second)
)

func init() {
	go func() {
		for {
			t := <-ticker.C
			checkFileLogPersistStatus(t)
		}
	}()
}

func checkFileLogPersistStatus(t time.Time) {
	checkTime := t.Unix()
	for i, _ := range checkFileLogger {
		fLength, err := checkFileLength(checkFileLogger[i].w.filename)
		switch {
		case checkFileLogger[i].lastCheckTime == 0:
			checkFileLogger[i].lastCheckFileLength = fLength
			checkFileLogger[i].lastCheckTime = checkTime
		case checkFileLogger[i].lastCheckTime > 0:
			if err != nil || fLength == checkFileLogger[i].lastCheckFileLength { //文件长度检查出错,或者未变化
				if checkFileLogger[i].w.lastWriteLogTime > checkFileLogger[i].lastCheckTime { //写日志时间大于检查时间
					checkFileLogger[i].lastCheckFileLength = 0 //重置检查信息
					checkFileLogger[i].lastCheckTime = 0
					checkFileLogger[i].w.intRotate(false)
				} else {
					checkFileLogger[i].lastCheckTime = checkTime //更新最后检查时间
				}
			}
		}
	}
}

func checkFileLength(fName string) (int64, error) {
	fLength := int64(0)
	f, err := os.Stat(fName)
	if err == nil && !f.IsDir() {
		fLength = f.Size()
	}
	return fLength, err
}
