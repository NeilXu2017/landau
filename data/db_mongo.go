package data

import (
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/NeilXu2017/landau/log"
	"github.com/NeilXu2017/landau/util"

	"github.com/globalsign/mgo"
)

type (
	//MongoDatabase mongo数据连接对象
	MongoDatabase struct {
		host       string
		port       int
		authSource string
		user       string
		password   string
		poolLimit  int
		timeout    int
		mgoLogger  string
	}
)

const (
	defaultAuthSource = "admin"
	defaultTimeout    = 5 //单位分钟
	defaultPoolLimit  = 4096
	defaultMgoLogger  = "main"
)

var (
	mgoDBConnectionPool sync.Map //key: mgoConnection value: *mgo.Session
)

// String 格式化输出
func (c *MongoDatabase) String() string {
	return fmt.Sprintf("%s:%d %s %s %s %d %d", c.host, c.port, c.authSource, c.user, c.password, c.poolLimit, c.timeout)
}

// NewMongoDatabase 构建MongoDatabase对象
func NewMongoDatabase(host string, port int, authSource string, user string, password string, poolLimit int, timeout int, logger string) *MongoDatabase {
	mgoDB := &MongoDatabase{
		host:       host,
		port:       port,
		authSource: authSource,
		user:       user,
		password:   password,
		poolLimit:  poolLimit,
		timeout:    timeout,
		mgoLogger:  logger,
	}
	return mgoDB
}

// NewMongoDatabase2 构建MongoDatabase对象
func NewMongoDatabase2(host string, port int, authSource string, user string, password string) *MongoDatabase {
	return NewMongoDatabase(host, port, authSource, user, password, defaultPoolLimit, defaultTimeout, defaultMgoLogger)
}

// NewMongoDatabase3 构建MongoDatabase对象
func NewMongoDatabase3(host string, port int) *MongoDatabase {
	return NewMongoDatabase(host, port, defaultAuthSource, "", "", defaultPoolLimit, defaultTimeout, defaultMgoLogger)
}

// GetMgoSession 获取连接MongoDB的SESSION
func (c *MongoDatabase) GetMgoSession() (*mgo.Session, error) {
	start := time.Now()
	mgoConnection := fmt.Sprintf("%v", c)
	if conn, ok := mgoDBConnectionPool.Load(mgoConnection); ok {
		mgoSession := conn.(*mgo.Session)
		return mgoSession.Copy(), nil
	}
	dialInfo := &mgo.DialInfo{
		Addrs:       []string{fmt.Sprintf("%s:%d", util.IPConvert(c.host, util.IPV6Bracket), c.port)},
		Timeout:     time.Duration(c.timeout) * time.Second,
		Source:      c.authSource,
		Username:    c.user,
		Password:    c.password,
		PoolLimit:   c.poolLimit,
		ReadTimeout: 5 * time.Minute,
	}
	mgoSession, err := mgo.DialWithInfo(dialInfo)
	if err == nil {
		mgoDBConnectionPool.Store(mgoConnection, mgoSession)
		return mgoSession.Copy(), nil

	}
	log.Error2(c.mgoLogger, "[MGO] [%s]\t DialInfo:[%v] Error:[%v]", time.Since(start), dialInfo, err)
	return nil, err
}

// IsExisted 检查目标db,collection 是否存在，1存在 0不存在，collection为空则仅仅检查db是否存在
func (c *MongoDatabase) IsExisted(db string, collection string) (int, error) {
	start := time.Now()
	if db == "" {
		return 0, fmt.Errorf("db is empty")
	}
	ms, err := c.GetMgoSession()
	if err != nil {
		return 0, err
	}
	defer ms.Close()
	dbs, dbErr := ms.DatabaseNames()
	if dbErr != nil {
		log.Error2(c.mgoLogger, "[MGO] [%s]\t DatabaseNames Error:[%v]", time.Since(start), dbErr)
	}
	for _, name := range dbs {
		if name == db {
			if collection == "" {
				return 1, nil
			}
			mgoD := ms.DB(db)
			cs, csErr := mgoD.CollectionNames()
			if csErr != nil {
				log.Error2(c.mgoLogger, "[MGO] [%s]\t CollectionNames(%s) Error:[%v]", time.Since(start), db, csErr)
				return 0, csErr
			}
			for _, cName := range cs {
				if cName == collection {
					return 1, nil
				}
			}
		}
	}
	return 0, nil
}

// Get 检索单条记录
func (c *MongoDatabase) Get(db, collection string, query, selector, result interface{}) error {
	start := time.Now()
	ms, err := c.GetMgoSession()
	if err != nil {
		return err
	}
	defer ms.Close()
	ms.SetMode(mgo.Monotonic, true)
	mgoCollections := ms.DB(db).C(collection)
	err = mgoCollections.Find(query).Select(selector).One(result)
	if err != nil {
		log.Error2(c.mgoLogger, "[MGO] [%s]\t Get(%s,%s) Query:%v Select:%v Result:%v Error:[%v]", time.Since(start), db, collection, query, selector, result, err)
		return err
	}
	log.Info2(c.mgoLogger, "[MGO] [%s]\t Get(%s,%s) Query:%v Select:%v Result:%v", time.Since(start), db, collection, query, selector, result)
	return nil
}

// Gets 检索多条记录
func (c *MongoDatabase) Gets(db, collection string, query, selector, result interface{}) error {
	start := time.Now()
	ms, err := c.GetMgoSession()
	if err != nil {
		return err
	}
	defer ms.Close()
	ms.SetMode(mgo.Monotonic, true)
	mgoCollections := ms.DB(db).C(collection)
	err = mgoCollections.Find(query).Select(selector).All(result)
	if err != nil {
		log.Error2(c.mgoLogger, "[MGO] [%s]\t Gets(%s,%s) Query:%v Select:%v Result:%v Error:[%v]", time.Since(start), db, collection, query, selector, result, err)
		return err
	}
	rowCounts := getArrayLen(reflect.ValueOf(result))
	log.Info2(c.mgoLogger, "[MGO] [%s]\t Gets(%s,%s) Query:%v Select:%v [ROW COUNT=%d]", time.Since(start), db, collection, query, selector, rowCounts)
	return nil
}

// Count 检索满足条件的记录个数
func (c *MongoDatabase) Count(db, collection string, query interface{}) (int, error) {
	start := time.Now()
	ms, err := c.GetMgoSession()
	if err != nil {
		return 0, err
	}
	defer ms.Close()
	ms.SetMode(mgo.Monotonic, true)
	mgoCollections := ms.DB(db).C(collection)
	rowCount, countErr := mgoCollections.Find(query).Count()
	if countErr != nil {
		log.Error2(c.mgoLogger, "[MGO] [%s]\t Count(%s,%s) Query:%v Error:[%v]", time.Since(start), db, collection, query, countErr)
		return 0, countErr
	}
	log.Info2(c.mgoLogger, "[MGO] [%s]\t Count(%s,%s) Query:%v Count:%d", time.Since(start), db, collection, query, rowCount)
	return rowCount, countErr
}

// GetPage 分页检索记录第一页从1开始，pageSize页大小
func (c *MongoDatabase) GetPage(db, collection string, query, selector, result interface{}, pageIndex, pageSize int) error {
	start := time.Now()
	ms, err := c.GetMgoSession()
	if err != nil {
		return err
	}
	defer ms.Close()
	ms.SetMode(mgo.Monotonic, true)
	mgoCollections := ms.DB(db).C(collection)
	skipPageCount := (pageIndex - 1) * pageSize
	err = mgoCollections.Find(query).Select(selector).Skip(skipPageCount).Limit(pageSize).All(result)
	if err != nil {
		log.Error2(c.mgoLogger, "[MGO] [%s]\t GetPage(%s,%s) Query:%v Select:%v Page Index:%d Page Size:%d Result:%v Error:[%v]", time.Since(start), db, collection, query, selector, pageIndex, pageSize, result, err)
		return err
	}
	rowCounts := getArrayLen(reflect.ValueOf(result))
	log.Info2(c.mgoLogger, "[MGO] [%s]\t GetPage(%s,%s) Query:%v Select:%v Page Index:%d Page Size:%d [ROW COUNT=%d]", time.Since(start), db, collection, query, selector, pageIndex, pageSize, rowCounts)
	return nil
}
