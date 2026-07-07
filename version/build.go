package version

import (
	"crypto/md5"
	"flag"
	"fmt"
	"os"
	"runtime"
	"strings"
)

var (
	releaseVersion string
	buildTime      string
	printVersion   = flag.Bool("version", false, "Show build version information")
)

// GetReleaseVersion 编译版本
func GetReleaseVersion() string {
	return releaseVersion
}

// GetBuildTime 编译时间
func GetBuildTime() string {
	return buildTime
}

// ShowVersion 显示编译版本信息
func ShowVersion() {
	if *printVersion {
		appFullPath := os.Args[0]
		if runtime.GOOS == "windows" {
			appFullPath = strings.Replace(appFullPath, "\\", "/", -1)
		}
		if i := strings.LastIndex(appFullPath, "/"); i > 0 {
			appFullPath = appFullPath[i+1:]
		}
		if buildTime == "" {
			if file, err := os.Stat(os.Args[0]); err == nil {
				buildTime = file.ModTime().Format("2006-01-02 15:04:05")
				if b, err := os.ReadFile(os.Args[0]); err == nil {
					releaseVersion = fmt.Sprintf("%x", md5.Sum(b))
				}
			}
		}
		fmt.Printf("Application:%s\nVersion:%s\nBuild Time:%s\n", appFullPath, releaseVersion, buildTime)
		os.Exit(0)
	}
}
