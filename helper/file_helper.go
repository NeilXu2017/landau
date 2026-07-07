package helper

import (
	"fmt"
	"os"
	"runtime"
	"strings"
)

//ConvertAbsolutePath 向上递归查询指定的文件并转换成绝对路径返回
func ConvertAbsolutePath(configFilePath string) string {
	if configFilePath[0:1] == "/" { //绝对路径
		return configFilePath
	}
	if runtime.GOOS == "windows" && len(configFilePath) > 1 && configFilePath[1:2] == ":" { //绝对路径
		return configFilePath
	}
	if workPath, err := os.Getwd(); err == nil {
		splitPathChar := "/"
		if runtime.GOOS == "windows" && strings.Index(workPath, "\\") > 0 {
			splitPathChar = "\\"
		}
		if exist, fullPath := existFile(configFilePath, workPath, splitPathChar); exist {
			return fullPath
		}
	}
	return configFilePath
}

func existFile(relativePathFile, workPath string, splitPathChar string) (bool, string) {
	checkFullPath := fmt.Sprintf("%s/%s", workPath, relativePathFile)
	_, err := os.Stat(checkFullPath)
	if err == nil {
		return true, checkFullPath
	}
	if workPath == "" || workPath == "/" {
		return false, ""
	}
	workPath = getParentDirectory(workPath, splitPathChar)
	return existFile(relativePathFile, workPath, splitPathChar)
}

func getParentDirectory(workPath string, splitPathChar string) string {
	iEndIndex := strings.LastIndex(workPath, splitPathChar)
	if iEndIndex == -1 || iEndIndex == 0 {
		return "/"
	}
	return workPath[0:iEndIndex]
}
