// +build windows

package log

import (
	"fmt"
)

func _getDiskUsagePercent(path string) (int, uint64, error) {
	return 1, 1024 * 1024 * 1024 * 20, fmt.Errorf("not implement")
}
