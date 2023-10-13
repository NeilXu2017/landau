package data

import (
	"fmt"
	"time"

	"github.com/NeilXu2017/landau/log"
	"github.com/NeilXu2017/landau/util"
	lock "github.com/bsm/redis-lock"
	"github.com/go-redis/redis"
)

type (
	//RedisDatabase Redis访问对象
	RedisDatabase struct {
		db           int
		password     string
		host         string
		port         int
		dialTimeout  int
		readTimeout  int
		writeTimeout int
		writeLog     bool
		logger       string
		poolSize     int
	}
	RedisOptionFunc func(*RedisDatabase)
)

const (
	defaultRedisWriteLog     = true
	defaultRedisLogger       = "main"
	defaultRedisDB           = 0
	defaultRedisPassword     = ""
	defaultRedisDialTimeout  = 5
	defaultRedisReadTimeout  = 3
	defaultRedisWriteTimeout = 3
)

func SetRedisPooSize(poolSize int) RedisOptionFunc {
	return func(c *RedisDatabase) {
		c.poolSize = poolSize
	}
}

func SetRedisPassword(password string) RedisOptionFunc {
	return func(c *RedisDatabase) {
		c.password = password
	}
}
func SetRedisDb(db int) RedisOptionFunc {
	return func(c *RedisDatabase) {
		c.db = db
	}
}
func SetRedisDialTimeout(dialTimeout int) RedisOptionFunc {
	return func(c *RedisDatabase) {
		c.dialTimeout = dialTimeout
	}
}
func SetRedisReadTimeout(readTimeout int) RedisOptionFunc {
	return func(c *RedisDatabase) {
		c.readTimeout = readTimeout
	}
}
func SetRedisWriteTimeout(writeTimeout int) RedisOptionFunc {
	return func(c *RedisDatabase) {
		c.writeTimeout = writeTimeout
	}
}
func SetRedisWriteLog(writeLog bool) RedisOptionFunc {
	return func(c *RedisDatabase) {
		c.writeLog = writeLog
	}
}
func SetRedisLoggName(loggerName string) RedisOptionFunc {
	return func(c *RedisDatabase) {
		c.logger = loggerName
	}
}

func NewRedisDatabase3(host string, port int, options ...RedisOptionFunc) *RedisDatabase {
	redisHelper := &RedisDatabase{
		host:         host,
		port:         port,
		db:           defaultRedisDB,
		password:     defaultRedisPassword,
		dialTimeout:  defaultRedisDialTimeout,
		readTimeout:  defaultRedisReadTimeout,
		writeTimeout: defaultRedisWriteTimeout,
		writeLog:     defaultRedisWriteLog,
		logger:       defaultRedisLogger,
	}
	for _, option := range options {
		option(redisHelper)
	}
	return redisHelper
}

// NewRedisDatabase 构建对象
func NewRedisDatabase(host string, port int, db int, password string, dialTimeout int, readTimeout int, writeTimeout int, writeLog bool, loggerName string) *RedisDatabase {
	redisHelper := &RedisDatabase{
		host:         host,
		port:         port,
		db:           db,
		password:     password,
		dialTimeout:  dialTimeout,
		readTimeout:  readTimeout,
		writeTimeout: writeTimeout,
		writeLog:     writeLog,
		logger:       loggerName,
	}
	return redisHelper
}

// NewRedisDatabase2 构建对象
func NewRedisDatabase2(host string, port int) *RedisDatabase {
	return NewRedisDatabase(host, port, defaultRedisDB, defaultRedisPassword, defaultRedisDialTimeout, defaultRedisReadTimeout, defaultRedisWriteTimeout, defaultRedisWriteLog, defaultRedisLogger)
}

// GetRedisClient 获取redisClient
func (c *RedisDatabase) GetRedisClient() (*redis.Client, error) {
	redisOptions := &redis.Options{
		Addr:         fmt.Sprintf("%s:%d", util.IPConvert(c.host, util.IPV6Bracket), c.port),
		Password:     c.password,
		DB:           c.db,
		DialTimeout:  time.Duration(c.dialTimeout) * time.Second,
		ReadTimeout:  time.Duration(c.readTimeout) * time.Second,
		WriteTimeout: time.Duration(c.writeTimeout) * time.Second,
		PoolSize:     c.poolSize,
	}
	client := redis.NewClient(redisOptions)
	_, err := client.Ping().Result()
	return client, err
}

// Set 存储
func (c *RedisDatabase) Set(key string, value string, TTL int) error {
	start := time.Now()
	client, err := c.GetRedisClient()
	defer func() {
		if err != nil {
			log.Error2(c.logger, "[Redis] [%s]\tSet Key:%s Value:%s Error:%v", time.Since(start), key, value, err)
		} else {
			if c.writeLog {
				log.Info2(c.logger, "[Redis] [%s]\tSet Key:%s Value:%s", time.Since(start), key, value)
			}
		}
	}()
	if err != nil {
		return err
	}
	defer client.Close()
	return client.Set(key, value, time.Duration(TTL)*time.Second).Err()
}

// Get 读取
func (c *RedisDatabase) Get(key string) (string, int, error) {
	start := time.Now()
	client, err := c.GetRedisClient()
	value := ""
	defer func() {
		if err != nil {
			log.Error2(c.logger, "[Redis] [%s]\tGet Key:%s Error:%v", time.Since(start), key, err)
		} else {
			if c.writeLog {
				log.Info2(c.logger, "[Redis] [%s]\tGet Key:%s Value:%v", time.Since(start), key, value)
			}
		}
	}()
	if err != nil {
		return "", 0, err
	}
	defer client.Close()
	value, _ = client.Get(key).Result()
	if err == redis.Nil {
		return value, 0, nil
	}
	if err != nil {
		return value, 0, err
	}
	return value, 1, err
}

// Lock 获取分布式锁，成功后确保调用 Unlock()释放锁
func (c *RedisDatabase) Lock(key string, lockTimeout int, retryCount int, retryDelay int) (*lock.Locker, error) {
	client, err := c.GetRedisClient()
	if err != nil {
		return nil, err
	}
	lockOpts := &lock.Options{
		LockTimeout: time.Duration(lockTimeout) * time.Second,
		RetryCount:  retryCount,
		RetryDelay:  time.Duration(retryDelay) * time.Millisecond,
	}
	return lock.Obtain(client, key, lockOpts)
}

// WaitLock 获取分布式锁,尝试在waitTimeout秒内获取锁
func (c *RedisDatabase) WaitLock(key string, lockTimeout int, waitTimeout int) (*lock.Locker, error) {
	retryDelay := time.Duration(100)
	retryCount := int(time.Second/(retryDelay*time.Millisecond)) * waitTimeout
	return c.Lock(key, lockTimeout, retryCount, int(retryDelay))
}
