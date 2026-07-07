package test

import (
	"fmt"
	"sync"
	"time"

	"github.com/NeilXu2017/landau/protocol"
)

// CheckServiceHelper 测试访问 zookeeper
func CheckServiceHelper() {
	path := fmt.Sprintf("//%d///", 0)
	zookeeperAddr := ""
	h := protocol.NewServiceAddrHelper2(zookeeperAddr, path)
	for i := 1; i < 11; i++ {
		ip, port, err := h.GetAddr()
		fmt.Printf("loop=%d\tIP=%s\tPort=%d\terror=%v\n", i, ip, port, err)
	}
}

// CheckServiceHelper2 测试
func CheckServiceHelper2() {
	path := "////"
	zs := map[string]string{
		"1": "",
		"2": "",
		"3": "",
		"4": "",
		"5": "",
	}
	wg := sync.WaitGroup{}
	loopCheckAddr := func(v *protocol.ServiceAddrHelper) {
		defer wg.Done()
		for i := 1; i < 11; i++ {
			ip, port, err := v.GetAddr()
			fmt.Printf("%s loop=%d\tIP=%s\tPort=%d\terror=%v\n", v.ZKey(), i, ip, port, err)
			time.Sleep(3 * time.Second)
		}
	}
	for region, zAddr := range zs {
		zsHelper := protocol.NewServiceAddrHelper3(zAddr, path, region)
		wg.Add(1)
		go loopCheckAddr(zsHelper)
	}
	wg.Wait()
}
