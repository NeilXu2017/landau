//go:build linux
// +build linux

package log

import (
	"fmt"
	"syscall"
)

func _getDiskUsagePercent(path string) (int, uint64, error) {
	usagePercent, allSize, freeSize := 0, uint64(1), uint64(1)
	fs := syscall.Statfs_t{}
	err := syscall.Statfs(path, &fs)
	if err == nil {
		allSize = fs.Blocks * uint64(fs.Bsize)
		freeSize = fs.Bfree * uint64(fs.Bsize)
		usagePercent = int((allSize - freeSize) * 100 / allSize)
	}
	if _monitorConfig.Debug {
		fmt.Printf("[LandauMonitorLog] directory:%s status: total size %d free size %d usage percent %d\n", path, allSize, freeSize, usagePercent)
	}
	return usagePercent, allSize, err
}
