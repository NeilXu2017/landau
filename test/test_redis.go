package test

import (
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"github.com/NeilXu2017/landau/data"
	"github.com/NeilXu2017/landau/log"
)

// CheckRedis 测试Redis
func CheckRedis() {
	redisDB := data.NewRedisDatabase("127.0.0.1", 6379, 0, "xxx", 5, 3, 3, true, "main")
	key := "hello_world"
	ttl := 300
	value, count, err := redisDB.Get(key)
	if err == nil {
		if count > 0 {
			fmt.Printf("Redis Get(%s)=%s\n", key, value)
		}
	}
	if count == 0 {
		fmt.Printf("Redis not found key:%s now set...\n", key)
		_ = redisDB.Set(key, fmt.Sprintf("Rand_%d", 1000+rand.Intn(1000)), ttl)
		checkValue, _, _ := redisDB.Get(key)
		fmt.Printf("Set and check:%s\n", checkValue)
	}
}

// CheckLockServer 测试 Locker
func CheckLockServer() {
	redisDB := data.NewRedisDatabase("127.0.0.1", 6379, 0, "xxx", 5, 3, 3, true, "main")
	if locker, err := redisDB.Lock("test_locker", 20, 10, 100); err == nil {
		log.Info("Acquire locker now")
		defer func() {
			_ = locker.Unlock()
			log.Info("Released locker now")
		}()
		v, _, _ := redisDB.Get("test_num")
		if v == "" {
			v = "0"
		}
		iV, _ := strconv.Atoi(v)
		iV -= rand.Intn(100)
		time.Sleep(15 * time.Second)
		_ = redisDB.Set("test_num", fmt.Sprintf("%d", iV), 900)
	}
}

// CheckLockClient 测试 Locker
func CheckLockClient() {
	redisDB := data.NewRedisDatabase("127.0.0.1", 6379, 0, "xxx", 30, 30, 30, true, "main")
	locker, err := redisDB.WaitLock("test_locker", 20, 15)
	if err != nil {
		fmt.Printf("%v", err)
		return
	}
	if err == nil {
		fmt.Printf("Accquired locker now\n")
		defer func() {
			_ = locker.Unlock()
			fmt.Printf("Released locker now\n")
		}()
		v, _, _ := redisDB.Get("test_num")
		if v == "" {
			v = "0"
		}
		iV, _ := strconv.Atoi(v)
		iV += rand.Intn(100)
		time.Sleep(3 * time.Second)
		_ = redisDB.Set("test_num", fmt.Sprintf("%d", iV), 900)
		fmt.Printf("Set Num %d\n", iV)
	}
}
