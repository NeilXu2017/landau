package data

import (
	"database/sql"
	"database/sql/driver"
	"github.com/NeilXu2017/landau/log"
	"sync"
	"time"
)

type (
	OptimizeConnection struct {
		dbConnection       string  //db.dbConnection
		badConnectionCount int     //发生无法连接的累积次数
		isBlocked          bool    //是否跳过
		pingCheckTime      int64   //上次DB ping的时间
		db                 *sql.DB //ping db 检测
	}
	OptimizeConnectionEventCallback func(dbConn, eventType string, t int64)
)

var (
	enableOptimizeConnection = false                                //是否开启连接优化,默认不开启
	badConnectionCount       = 20                                   //判定DB连接出现问题的累积 错误数目,最小值3次
	pingDBInterval           = int64(3)                             //DB ping 检测间隔时间
	optimizeConnections      = make(map[string]*OptimizeConnection) //key: db.dbConnection value: pointer of struct OptimizeConnection
	optimizeConnectionSync   = sync.RWMutex{}                       //同步访问
	pingCheckRunning         = false                                //prevent repeat call
	callback                 OptimizeConnectionEventCallback        //event call back
	traceOptimizeAction      = false                                //trace log
)

func SetOptimizeConnectionTrace(trace bool)                                { traceOptimizeAction = trace }
func SetOptimizeConnectionEventCallback(c OptimizeConnectionEventCallback) { callback = c }
func SetEnableOptimizeConnection(enable bool) {
	enableOptimizeConnection = enable
	if enableOptimizeConnection {
		if pingCheckRunning == false {
			pingCheckRunning = true
			go dbPingCheck()
			if traceOptimizeAction {
				log.Info("[SetEnableOptimizeConnection] start dbPingCheck go routine.")
			}
		}
	} else {
		if pingCheckRunning == true {
			pingCheckRunning = false
			optimizeConnectionSync.Lock()
			defer optimizeConnectionSync.Unlock()
			optimizeConnections = make(map[string]*OptimizeConnection)
			if traceOptimizeAction {
				log.Info("[SetEnableOptimizeConnection] stop dbPingCheck go routine.")
			}
		}
	}
}
func SetBadConnectionCount(badCount int) {
	if badCount >= 3 {
		badConnectionCount = badCount
	}
}
func SetDbPingCheckInterval(pingInterval int64) {
	if pingInterval > 0 {
		pingDBInterval = pingInterval
	}
}

func dbConnectionOk(dbConnection string) {
	if enableOptimizeConnection {
		optimizeConnectionSync.RLock()
		c, deleteConn := optimizeConnections[dbConnection]
		optimizeConnectionSync.RUnlock()
		if deleteConn {
			dbConn, eventType, t := c.dbConnection, "resume", time.Now().Unix()
			optimizeConnectionSync.Lock()
			defer optimizeConnectionSync.Unlock()
			delete(optimizeConnections, dbConnection)
			if callback != nil {
				go callback(dbConn, eventType, t)
			}
			if traceOptimizeAction {
				log.Info("[dbConnectionOk] resume db connection dbConn:{%s}.", dbConnection)
			}
		}
	}
}

func dbConnectionError(dbConnection string, err error, db *sql.DB) {
	if enableOptimizeConnection && db != nil {
		if err.Error() == driver.ErrBadConn.Error() {
			optimizeConnectionSync.Lock()
			defer optimizeConnectionSync.Unlock()
			if v, ok := optimizeConnections[dbConnection]; ok {
				v.badConnectionCount++
				if v.isBlocked == false && v.badConnectionCount >= badConnectionCount {
					v.isBlocked = true
					log.Info("[dbConnectionError] %s block connection now", dbConnection)
					dbConn, eventType, t := v.dbConnection, "block", time.Now().Unix()
					if callback != nil {
						go callback(dbConn, eventType, t)
					}
				} else {
					if traceOptimizeAction {
						log.Info("[dbConnectionError] badConnectionCount=%d maxbadConnectionCount=%d dbKey={%s}.", v.badConnectionCount, badConnectionCount, dbConnection)
					}
				}
			} else {
				optimizeConnections[dbConnection] = &OptimizeConnection{
					dbConnection:       dbConnection,
					isBlocked:          false,
					badConnectionCount: 1,
					db:                 db,
				}
			}
		}
	}
}

func skipDbAccessDueErrBadConnection(dbConnection string) (bool, error) {
	skipDbAccessDue := false
	var err error
	optimizeConnectionSync.RLock()
	defer optimizeConnectionSync.RUnlock()
	if v, ok := optimizeConnections[dbConnection]; ok && v.isBlocked {
		skipDbAccessDue = true
		err = driver.ErrBadConn
	}
	return skipDbAccessDue, err
}

func dbPingCheck() {
	for {
		wg, t, pingOk, syncCheck := sync.WaitGroup{}, time.Now().Unix(), []string{}, sync.Mutex{}
		_check := func(key string, db *sql.DB) {
			defer func() {
				if err := recover(); err != nil {
					log.Error("[_check] recover:%v", err)
				}
				wg.Done()
			}()
			pingTime := time.Now()
			err := db.Ping()
			if err == nil {
				syncCheck.Lock()
				pingOk = append(pingOk, key)
				syncCheck.Unlock()
			}
			if traceOptimizeAction {
				log.Info("[db.Ping()] [%s] {%s} ping result:%v", time.Since(pingTime), key, err)
			}
		}
		optimizeConnectionSync.Lock()
		for _, d := range optimizeConnections {
			if t-d.pingCheckTime >= pingDBInterval {
				key := d.dbConnection
				d.pingCheckTime = t
				wg.Add(1)
				go _check(key, d.db)
			}
		}
		optimizeConnectionSync.Unlock()
		wg.Wait()
		if len(pingOk) > 0 {
			eventTime, eventType := time.Now().Unix(), "resume"
			optimizeConnectionSync.Lock()
			for _, k := range pingOk {
				delete(optimizeConnections, k)
				log.Info("[dbPingCheck] %s ping OK,resume connection now", k)
				if callback != nil {
					go callback(k, eventType, eventTime)
				}
			}
			optimizeConnectionSync.Unlock()
		}
		time.Sleep(time.Duration(pingDBInterval) * time.Second)
		if pingCheckRunning == false {
			break
		}
	}
}
