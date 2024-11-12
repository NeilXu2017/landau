package log

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime/debug"
	"sort"
	"strings"
	"time"
)

type (
	_LogFileMonitorConfig struct {
		DisableLogFileMonitor bool `json:"disable_log_file_monitor"` //禁止监控
		CheckInterval         int  `json:"check_interval"`           //检查周期 单位秒
		CleanUsagePercent     int  `json:"clean_usage_percent"`      //磁盘空间利用率,大于此阀值,清理log文件
		ReleaseSizePercent    int  `json:"release_size_percent"`     //清理时,尽量释放的空虚比例
		Debug                 bool `json:"debug"`                    //Trace开关
		CleanDailyMode        bool `json:"clean_daily_mode"`         //daily 模式监控日志
		KeepDayCount          int  `json:"keep_day_count"`           //保留最近今天的日志,不包含当日
	}
	_LogFileMonitorPath struct {
		FileName string //日志文件名称
		Path     string //日志文件所在目录
	}
	_CleanFileInfo struct {
		Name       string
		Size       uint64
		ModifyTime uint64
	}
	_SortCleanFileInfo []_CleanFileInfo
)

const (
	_minCheckInterval      = 10 //最小检查间隔周期
	_maxUsagePercent       = 95 //最大磁盘使用量
	_minReleaseSizePercent = 5  //最小释放比例
)

var (
	_monitorConfig = _LogFileMonitorConfig{CheckInterval: 300, CleanUsagePercent: 85, ReleaseSizePercent: 20, KeepDayCount: 3} //默认值: 5分钟检查一次 磁盘空间使用率 >=85 需要清理日志文件
	_monitorFiles  []_LogFileMonitorPath
)

func (a _SortCleanFileInfo) Len() int      { return len(a) }
func (a _SortCleanFileInfo) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a _SortCleanFileInfo) Less(i, j int) bool {
	if a[i].ModifyTime == a[j].ModifyTime {
		return a[i].Size > a[j].Size
	}
	return a[i].ModifyTime < a[j].ModifyTime
}

func _initMonitorConfig(c _LogFileMonitorConfig, logFileNames []string) {
	_monitorConfig.DisableLogFileMonitor = c.DisableLogFileMonitor
	_monitorConfig.Debug = c.Debug
	_monitorConfig.CleanDailyMode = c.CleanDailyMode
	if c.CheckInterval >= _minCheckInterval {
		_monitorConfig.CheckInterval = c.CheckInterval
	}
	if c.CleanUsagePercent > 0 && c.CleanUsagePercent < _maxUsagePercent {
		_monitorConfig.CleanUsagePercent = c.CleanUsagePercent
	}
	if c.ReleaseSizePercent > _minReleaseSizePercent {
		_monitorConfig.ReleaseSizePercent = c.ReleaseSizePercent
	}
	if c.KeepDayCount > 0 {
		_monitorConfig.KeepDayCount = c.KeepDayCount
	}
	currentPath := _getCurrentPath()
	for _, f := range logFileNames {
		paths := strings.Split(f, "/")
		mf := _LogFileMonitorPath{FileName: paths[len(paths)-1]}
		switch {
		case paths[0] == ".":
			mf.Path = fmt.Sprintf("%s/%s", currentPath, strings.Join(paths[1:len(paths)-1], "/"))
		case f[0:1] != "/":
			mf.Path = fmt.Sprintf("%s/%s", currentPath, strings.Join(paths[:len(paths)-1], "/"))
		default:
			mf.Path = strings.Join(paths[:len(paths)-1], "/")
		}
		mf.Path = strings.ReplaceAll(mf.Path, "//", "/")
		_monitorFiles = append(_monitorFiles, mf)
	}
	fmt.Printf("[MonitorLog] config:%v %v\n", _monitorConfig, _monitorFiles)
	if !c.DisableLogFileMonitor {
		go _loopMonitorJob()
	}
}

func _loopMonitorJob() {
	for {
		time.Sleep(time.Second * time.Duration(_monitorConfig.CheckInterval))
		_monitor()
	}
}

func _monitor() {
	defer func() {
		if err := recover(); err != nil {
			fmt.Print(err)
			debug.PrintStack()
		}
		if _monitorConfig.Debug {
			fmt.Printf("[MonitorLog] clean job complete\n")
		}
	}()
	if _monitorConfig.CleanDailyMode {
		for _, mf := range _monitorFiles {
			currentFileName := mf.FileName
			files, _ := os.ReadDir(mf.Path)
			for _, f := range files {
				fName := f.Name()
				historyLogFileName := regexp.MustCompile(fmt.Sprintf(`^%s\d{4}-\d{2}-d{2}$`, currentFileName))
				if currentFileName != fName && historyLogFileName.MatchString(fName) { //不是当前日志文件, 是历史文件
					execCmd(fmt.Sprintf("tar czvf %s/%s.tar.z %s/%s", mf.Path, fName, mf.Path, fName))
					_ = os.Remove(fmt.Sprintf("%s/%s", mf.Path, fName))
				}
			}
		}
		return
	}
	usagePercents, cleanPath, releaseSizes := make(map[string]int), make(map[string]string), make(map[string]uint64)
	//同批次清理
	for _, f := range _monitorFiles {
		v, ok := usagePercents[f.Path]
		if !ok {
			if vv, allSize, err := _getDiskUsagePercent(f.Path); err == nil {
				v = vv
				releaseSizes[f.Path] = allSize * uint64(_monitorConfig.ReleaseSizePercent) / 100
			}
			usagePercents[f.Path] = v
		}
		if v >= _monitorConfig.CleanUsagePercent {
			cleanPath[f.FileName] = f.Path
		}
	}
	if _monitorConfig.Debug {
		fmt.Printf("[MonitorLog] matched clean path info:%v need released info:%v\n", cleanPath, releaseSizes)
	}
	//清理log
	for fName, fPath := range cleanPath {
		if cleanSize, ok := releaseSizes[fPath]; ok {
			files, _ := os.ReadDir(fPath)
			var cleanFiles []_CleanFileInfo
			for _, f := range files {
				if !f.IsDir() {
					if ff, err := f.Info(); err == nil {
						if ff.Name() != fName && strings.Contains(ff.Name(), fName) { //不是当前日志文件, 是历史文件
							cleanFiles = append(cleanFiles, _CleanFileInfo{
								Name:       f.Name(),
								Size:       uint64(ff.Size()),
								ModifyTime: uint64(ff.ModTime().Unix()),
							})
						}
					}
				}
			}
			sort.Sort(_SortCleanFileInfo(cleanFiles))
			if _monitorConfig.Debug {
				fmt.Printf("[MonitorLog] sort clean files:%v\n", cleanFiles)
			}
			releasedSize := uint64(0)
			for _, f := range cleanFiles {
				_ = os.Remove(fmt.Sprintf("%s/%s", fPath, f.Name))
				releasedSize = releasedSize + f.Size
				if releasedSize >= cleanSize {
					break
				}
			}
		}
	}
}

func _getCurrentPath() string {
	if file, err := exec.LookPath(os.Args[0]); err == nil {
		if path, err := filepath.Abs(file); err == nil {
			i := strings.LastIndex(path, "/")
			if i < 0 {
				i = strings.LastIndex(path, "\\")
			}
			if i > 0 {
				return path[0 : i+1]
			}
		}
	}
	return ""
}

func _getKeptDays() []*regexp.Regexp {
	var m []*regexp.Regexp
	t := time.Now()
	for i := 1; i <= _monitorConfig.KeepDayCount; i++ {
		dayId := t.Add(time.Duration(-24*i) * time.Hour).Format("2006-01-02")
		m = append(m, regexp.MustCompile(fmt.Sprintf("^.*%s$", dayId)))
	}
	return m
}

func execCmd(cmd string) {
	var c *exec.Cmd
	c = exec.Command("/bin/bash", "-c", cmd)
	_ = c.Run()
}
