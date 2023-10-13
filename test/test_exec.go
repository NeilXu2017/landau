package test

import (
	"bytes"
	"fmt"
	"github.com/NeilXu2017/landau/util"
	"math"
	"math/rand"
	"runtime"
	"strconv"
	"sync"
	"time"
)

// CheckExecCmd 测试 Exec执行外部命令
func CheckExecCmd() {
	fmt.Println("CHECK EXEC")
	cmd := "go version"
	_, _ = util.ExecCmd(cmd)
	cmdOutPut, _ := util.ExecCmd2(cmd, []string{"version", "12"}, nil)
	fmt.Printf("%s\n", cmdOutPut)

	cmd = "./test-exec -c 0 -e hello"
	_, _ = util.ExecCmd(cmd)
	cmd = "./test-exec -c 0"
	_, _ = util.ExecCmd(cmd)
	cmd = "./test-exec -c 3"
	_, _ = util.ExecCmd(cmd)
	cmd = "./test-exec -c 12 -e Hello"
	_, _ = util.ExecCmd(cmd)
}

func CheckThread() {
	wg := sync.WaitGroup{}
	f := func() {
		defer wg.Done()
		id := getGID()
		for i := 0; i < 10000; i++ {
			s, y := rand.Intn(10), rand.Intn(100)
			d := (s+1)*731 + (y+1)*387
			math.Sqrt(float64(d))
			math.Acosh(float64(d))
			math.Atan(float64(d))
			time.Sleep(time.Duration((s+1)*17) * time.Millisecond)
			math.Log10(float64(d))
			nId := getGID()
			if nId != id {
				fmt.Printf("开始时在携程：%d, 当前在：%d\n", id, nId)
			}
		}
	}
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go f()
	}
	wg.Wait()
	fmt.Println("Check completed")
}
func getGID() uint64 {
	b := make([]byte, 64)
	b = b[:runtime.Stack(b, false)]
	b = bytes.TrimPrefix(b, []byte("goroutine "))
	b = b[:bytes.IndexByte(b, ' ')]
	n, _ := strconv.ParseUint(string(b), 10, 64)
	return n
}
