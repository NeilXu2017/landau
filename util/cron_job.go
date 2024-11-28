package util

import (
	"context"
	"math/rand"
	"reflect"
	"sync"
	"time"

	"github.com/NeilXu2017/landau/log"
	"github.com/robfig/cron"
)

type (
	//SingletonCronTask 定时任务定义, 调用次序: CallbackFunc 有则调用,无则检查 Instance 有则尝试调用, 无则检查 全局 Instance (数组外传递)
	SingletonCronTask struct {
		Name         string      //Job 名称
		Enable       bool        //是否启动
		Schedule     string      //定时安排
		FuncName     string      //执行的函数名称
		Instance     interface{} //通过对象实例的实现函数, 必须是对象的指针
		CallbackFunc func()      //实现函数. 直接调用,不通过 Instance对象实例
		Immediate    bool        //是否立即执行
	}
	CronJobManager struct {
		JobRunningState           map[string]bool
		SyncJobRunningStateLocker sync.RWMutex
	}
)

const shutdownPollIntervalMax = 500 * time.Millisecond

var (
	engine            *cron.Cron
	s                 *CronJobManager
	ScheduledJobCount = uint(0)
)

func (c *CronJobManager) GetJobRunningState(jobName string) bool {
	c.SyncJobRunningStateLocker.RLock()
	defer c.SyncJobRunningStateLocker.RUnlock()
	return c.JobRunningState[jobName]
}

func (c *CronJobManager) SetJobRunningState(jobName string, runningState bool) {
	c.SyncJobRunningStateLocker.Lock()
	defer c.SyncJobRunningStateLocker.Unlock()
	c.JobRunningState[jobName] = runningState
}

func (c *CronJobManager) IsAnyJobRunningState() bool {
	c.SyncJobRunningStateLocker.RLock()
	defer c.SyncJobRunningStateLocker.RUnlock()
	for _, running := range c.JobRunningState {
		if running {
			return true
		}
	}
	return false
}

func StartCronJob(p interface{}, jobs []SingletonCronTask) {
	if len(jobs) > 0 {
		for _, t := range jobs {
			if t.Enable {
				if engine == nil {
					engine = cron.New()
					s = &CronJobManager{JobRunningState: make(map[string]bool), SyncJobRunningStateLocker: sync.RWMutex{}}
				}
				n, fN, fI, fCallback := t.Name, t.FuncName, t.Instance, t.CallbackFunc
				jobFuncProxy := func() {
					if s.GetJobRunningState(n) {
						log.Info("[CronJobManager] [%s] previous job is running,skip once.", n)
						return
					}
					start := time.Now()
					defer func() {
						log.Info("[CronJobManager] [%s] [%s] complete.", n, time.Since(start))
						s.SetJobRunningState(n, false)
					}()
					s.SetJobRunningState(n, true)
					switch {
					case fCallback != nil:
						fCallback()
					case fI != nil:
						if v := reflect.ValueOf(fI); v.IsValid() {
							if vv := v.MethodByName(fN); vv.IsValid() {
								vv.Call(nil)
							} else {
								log.Error("[CronJobManager] [%s] no method %s in %v", n, fN, fI)
							}
						} else {
							log.Error("[CronJobManager] [%s] Instance:%v is not valid", n, fI)
						}
					case p != nil:
						if v := reflect.ValueOf(p); v.IsValid() {
							if vv := v.MethodByName(fN); vv.IsValid() {
								vv.Call(nil)
							} else {
								log.Error("[CronJobManager] [%s] no method %s in %v", n, fN, p)
							}
						} else {
							log.Error("[CronJobManager] [%s] interface:%v is not valid", n, p)
						}
					default:
						log.Error("[CronJobManager] [%s] missing job function.", n)
					}
				}
				_ = engine.AddFunc(t.Schedule, jobFuncProxy)
				ScheduledJobCount++
				log.Info("[CronJobManager] %s ADD,schedule:%s", t.Name, t.Schedule)
				if t.Immediate {
					go jobFuncProxy()
				}
			} else {
				log.Info("[CronJobManager] %s SKIP", t.Name)
			}
		}
		if engine != nil {
			engine.Start()
		}
	}
}

func CronJobShutdown(ctx context.Context) error {
	if engine != nil {
		engine.Stop()
		pollIntervalBase := time.Millisecond
		nextPollInterval := func() time.Duration {
			interval := pollIntervalBase + time.Duration(rand.Intn(int(pollIntervalBase/10)))
			pollIntervalBase *= 2
			if pollIntervalBase > shutdownPollIntervalMax {
				pollIntervalBase = shutdownPollIntervalMax
			}
			return interval
		}
		timer := time.NewTimer(nextPollInterval())
		defer timer.Stop()
		for {
			if !s.IsAnyJobRunningState() { //等待 cron job 完成
				return nil
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-timer.C:
				timer.Reset(nextPollInterval())
			}
		}
	}
	return nil
}
