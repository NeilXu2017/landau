package util

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/NeilXu2017/landau/log"
	"github.com/google/uuid"
)

var (
	_traceLogBeforeRunning bool
)

func SetTraceLogBeforeRunning(bTraceLog bool) {
	_traceLogBeforeRunning = bTraceLog
}

// ExecCmd 执行命令
func ExecCmd(cmd string) (string, error) {
	return ExecCmd3(cmd, []string{}, nil, nil)
}

// ExecCmd2 执行命令 cmd 命令字符串 replacedLogStr 替换记录的敏感信息 stdin 重定向标准输入
func ExecCmd2(cmd string, replacedLogStr []string, stdin io.Reader) (string, error) {
	return ExecCmd3(cmd, replacedLogStr, stdin, nil)
}

func ExecCmd3(cmd string, replacedLogStr []string, stdin io.Reader, formatCmdOut func(string) string) (string, error) {
	return ExecCmd4(cmd, replacedLogStr, stdin, formatCmdOut, log.IsEnableCategoryInfoLog("ExecCMD"))
}

func ExecCmdAlwaysLog2(cmd string, replacedLogStr []string, stdin io.Reader, formatCmdOut func(string) string) (string, error) {
	return ExecCmd4(cmd, replacedLogStr, stdin, formatCmdOut, true)
}

func ExecCmdAlwaysLog(cmd string) (string, error) {
	return ExecCmd4(cmd, []string{}, nil, nil, true)
}

func ExecCmd4(cmd string, replacedLogStr []string, stdin io.Reader, formatCmdOut func(string) string, enableLog bool) (string, error) {
	return ExecCmd5(cmd, replacedLogStr, stdin, formatCmdOut, enableLog, true)
}

func ExecCmd5(cmd string, replacedLogStr []string, stdin io.Reader, formatCmdOut func(string) string, enableLog bool, mergeStderr bool) (string, error) {
	start := time.Now()
	var c *exec.Cmd
	if runtime.GOOS == "windows" {
		cmdSlice := strings.Split(cmd, " ")
		szCmd := cmdSlice[0]
		args := cmdSlice[1:]
		c = exec.Command(szCmd, args...)
	} else {
		c = exec.Command("/bin/bash", "-c", cmd)
	}
	var out, stderrOut bytes.Buffer
	c.Stdout = &out
	c.Stdin = stdin
	c.Stderr = &stderrOut
	logCmd := cmd
	for _, searchedStr := range replacedLogStr {
		logCmd = strings.Replace(logCmd, searchedStr, "***", -1)
	}
	cmdMatchID := ""
	if _traceLogBeforeRunning {
		cmdMatchID = uuid.NewString()
		if enableLog {
			log.Info("[Exec CMD] Before Running(%s):\t%s", cmdMatchID, logCmd)
		}
	}
	err := c.Run()
	outStr := out.String()
	stdErrStr := stderrOut.String()
	if outStr == "" && mergeStderr { //stdout 无内容时，如果开启了合并 stderr,有内容或者有 err 输出到out
		if stdErrStr != "" {
			outStr = stdErrStr
		} else if err != nil {
			outStr = fmt.Sprintf("%v", err)
		}
	}
	if err != nil || stdErrStr != "" || enableLog {
		logOutStr := outStr
		if formatCmdOut != nil {
			logOutStr = formatCmdOut(outStr)
		}
		if err != nil || stdErrStr != "" {
			log.Warn("[Exec CMD%s]\t%s\tcmd:{%s}\tstdout:{%s}\tstderr:{%s}\t%v", cmdMatchID, time.Since(start), logCmd, logOutStr, stdErrStr, err)
		} else {
			log.Info("[Exec CMD%s]\t%s\t%s\t%s\t%v", cmdMatchID, time.Since(start), logCmd, logOutStr, err)
		}
	}
	return outStr, err
}
