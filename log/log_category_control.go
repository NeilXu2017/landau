package log

var (
	_CategoryInfoLogLevel = map[string]int{
		"HTTP":    1,
		"SQL":     1,
		"ExecCMD": 1,
	}
)

// IsEnableCategoryInfoLog 检测Info日志是否需要记录
func IsEnableCategoryInfoLog(category string) bool {
	return _CategoryInfoLogLevel[category] == 1
}

// SetCategoryInfoLogOption 设置Info日志记录选项
func SetCategoryInfoLogOption(category string, enable bool) {
	infoLeve := 1
	if !enable {
		infoLeve = 0
	}
	_SetCategoryInfoLog(category, infoLeve)
}

func _SetCategoryInfoLog(category string, infoLeve int) {
	_CategoryInfoLogLevel[category] = infoLeve
}
