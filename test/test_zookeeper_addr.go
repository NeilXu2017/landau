package test

import (
	"fmt"
	"sync"
	"time"

	"github.com/NeilXu2017/landau/protocol"
)

// CheckServiceHelper 测试访问 zookeeper
func CheckServiceHelper() {
	path := fmt.Sprintf("/NS/region%d/database_ipv6/ipv6/uaccount_rpcV6", 1000001)
	zookeeperAddr := "[2002:ac12:b0ab::1]:2181,[2002:ac12:b0ac::1]:2181,[2002:ac12:b4ab::1]:2181,[2002:ac12:b4ac::1]:2181,[2002:ac12:b4ad::1]:2181"
	h := protocol.NewServiceAddrHelper2(zookeeperAddr, path)
	for i := 1; i < 11; i++ {
		ip, port, err := h.GetAddr()
		fmt.Printf("loop=%d\tIP=%s\tPort=%d\terror=%v\n", i, ip, port, err)
	}
}

// CheckServiceHelper2 测试
func CheckServiceHelper2() {
	path := "/NS/umonitor2/set1/access"
	zs := map[string]string{
		"cn-bj2": "10.68.128.182:2181,10.68.128.183:2181,10.68.128.197:2181,10.68.132.143:2181,10.69.164.164:2181,10.69.164.165:2181,10.69.164.166:2181,10.69.164.167:2181,172.18.210.22:2181,172.18.213.118:2181,172.18.210.23:2181",
		"cn-gd":  "10.67.184.88:2181,10.67.184.89:2181,10.67.184.90:2181,10.67.184.91:2181,10.67.184.84:2181",
		"hk":     "10.68.68.108:2181,10.68.68.109:2181,10.68.68.110:2181,172.21.50.3:2181,172.21.50.23:2181",
		"us-ca":  "10.70.40.40:2181,10.70.40.41:2181,10.70.40.42:2181,10.70.40.43:2181,10.70.40.44:2181",
		"cn-sh2": "10.66.170.145:2181,10.66.170.148:2181,10.66.172.150:2181,10.66.170.149:2181,10.66.172.151:2181",
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
