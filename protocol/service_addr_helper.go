package protocol

import (
	"fmt"
	"github.com/samuel/go-zookeeper/zk"
	"math/rand"
	"sync"

	"github.com/NeilXu2017/landau/log"
	"github.com/NeilXu2017/landau/zookeeper"
)

type (
	//BalanceMode 获取服务地址方式
	BalanceMode int
	ipPortPair  struct {
		IP   string
		Port uint32
	}
	serviceAddress struct {
		address        []ipPortPair
		nextUsingIndex int
	}
	//ServiceAddrHelper 获取服务地址封装
	ServiceAddrHelper struct {
		zooKeeperAddr     string
		zooKeeperPath     string
		balanceMode       BalanceMode
		logger            string
		printDebugAddress bool
		regionId          string
	}
)

const (
	//RandomServer 随机
	RandomServer BalanceMode = iota
	//LastServer 取最后一个
	LastServer
	//PollingServer 轮流负载模式
	PollingServer
)

var (
	defaultBalanceMode         = PollingServer
	defaultServiceHelperLogger = "main"
	defaultPrintDebugAddress   = true
	serviceAddressPool         sync.Map //*serviceAddress
	serviceAddressRWMutexPool  sync.Map //*RWMutex 分 zone key 同步访问
)

// NewServiceAddrHelper 构建 ServiceAddrHelper对象
func NewServiceAddrHelper(zooKeeperAddr string, zooKeeperPath string, balanceMode BalanceMode, loggerName string, printDebug bool) *ServiceAddrHelper {
	return NewServiceAddrHelperEx(zooKeeperAddr, zooKeeperPath, balanceMode, loggerName, printDebug, "")
}

// NewServiceAddrHelper2 构建 ServiceAddrHelper对象
func NewServiceAddrHelper2(zooKeeperAddr string, zooKeeperPath string) *ServiceAddrHelper {
	return NewServiceAddrHelperEx(zooKeeperAddr, zooKeeperPath, defaultBalanceMode, defaultServiceHelperLogger, defaultPrintDebugAddress, "")
}

func NewServiceAddrHelper3(zooKeeperAddr string, zooKeeperPath string, regionId string) *ServiceAddrHelper {
	return NewServiceAddrHelperEx(zooKeeperAddr, zooKeeperPath, defaultBalanceMode, defaultServiceHelperLogger, defaultPrintDebugAddress, regionId)
}

func NewServiceAddrHelperEx(zooKeeperAddr string, zooKeeperPath string, balanceMode BalanceMode, loggerName string, printDebug bool, regionId string) *ServiceAddrHelper {
	s := &ServiceAddrHelper{
		zooKeeperAddr:     zooKeeperAddr,
		zooKeeperPath:     zooKeeperPath,
		balanceMode:       balanceMode,
		logger:            loggerName,
		printDebugAddress: printDebug,
		regionId:          regionId,
	}
	zKey := s.ZKey()
	if _, ok := serviceAddressRWMutexPool.Load(zKey); !ok {
		serviceAddressRWMutexPool.Store(zKey, &sync.RWMutex{}) //保存同步对象
	}
	return s
}

func getAddr(s *serviceAddress, balanceMode BalanceMode) (string, uint32, error) {
	usingIndex := s.nextUsingIndex
	if usingIndex >= len(s.address) {
		log.Info("[ServiceAddrHelper] [getAddr] Unexpected index value,change to zero now. Detail index:%d, address length:%d, serviceAddress:%+v BalanceMode:%d", usingIndex, len(s.address), s, balanceMode)
		usingIndex = 0
	}
	switch balanceMode {
	case RandomServer:
		s.nextUsingIndex = rand.Intn(len(s.address))
	case PollingServer:
		s.nextUsingIndex++
		if s.nextUsingIndex >= len(s.address) {
			s.nextUsingIndex = 0
		}
	}
	return s.address[usingIndex].IP, s.address[usingIndex].Port, nil
}

// ZKey 多地域支持
func (c *ServiceAddrHelper) ZKey() string {
	return fmt.Sprintf("%s%s", c.regionId, c.zooKeeperPath)
}

// GetAddr 获取服务地址
func (c *ServiceAddrHelper) GetAddr() (string, uint32, error) {
	zKey := c.ZKey()
	zKeyMutex, ok := serviceAddressRWMutexPool.Load(zKey)
	if !ok {
		return "", 0, fmt.Errorf("no found RWMutex object zKey=%s", zKey)
	}
	zKeySyncMutex := zKeyMutex.(*sync.RWMutex) //分 ZKey() 同步锁
	retIp, retPort, bFound := "", uint32(0), false
	zKeySyncMutex.RLock() //获取读锁
	if sAddress, ok := serviceAddressPool.Load(zKey); ok {
		s := sAddress.(*serviceAddress)
		retIp, retPort, _ = getAddr(s, c.balanceMode)
		bFound = true
	}
	zKeySyncMutex.RUnlock()
	if bFound {
		return retIp, retPort, nil
	}
	zKeySyncMutex.Lock() //获取写锁
	defer zKeySyncMutex.Unlock()
	if sAddress, ok := serviceAddressPool.Load(zKey); ok { //获得锁后,再次检查  serviceAddressPool 有没有缓存地址
		s := sAddress.(*serviceAddress)
		return getAddr(s, c.balanceMode)
	}
	zc := zookeeper.NewZConnector2(c.zooKeeperAddr)
	zConn, err := zc.GetConn()
	if err != nil {
		return "", 0, err
	}
	children, _, ch, watchErr := zConn.ChildrenW(c.zooKeeperPath)
	if watchErr != nil {
		return "", 0, watchErr
	}
	go func() {
		if c.printDebugAddress {
			log.Info2(c.logger, "[ServiceAddrHelper] WatchDog Monitor(region:%s node path:%s address:%s) start", c.regionId, c.zooKeeperPath, c.zooKeeperAddr)
		}
		for {
			ev := <-ch
			log.Info2(c.logger, "[ServiceAddrHelper] WatchDog Monitor(region:%s node path:%s address:%s) occur event:%v", c.regionId, c.zooKeeperPath, c.zooKeeperAddr, ev)
			switch ev.Type {
			case zk.EventNotWatching, zk.EventNodeDataChanged, zk.EventNodeChildrenChanged, zk.EventNodeCreated, zk.EventNodeDeleted:
				serviceAddressPool.Delete(zKey)
				return
			}
			if ev.State != zk.StateHasSession {
				serviceAddressPool.Delete(zKey)
				return
			}
		}
	}()
	var addr []ipPortPair
	for _, childPath := range children {
		childNodePath := c.zooKeeperPath + "/" + childPath
		bAddr, _, err := zc.GetNode(childNodePath)
		if err != nil {
			log.Error2(c.logger, "[ServiceAddrHelper] GetNode region:%s path:%s error:%v", c.regionId, childNodePath, err)
			continue
		}
		//TODO zk 存储数据解析
		if c.printDebugAddress {
			log.Info2(c.logger, "[ServiceAddrHelper] Address:%s region:%s NodePath:%s Addr:%+v", c.zooKeeperAddr, c.regionId, childNodePath, bAddr)
		}
		addr = append(addr, ipPortPair{}) //TODO wait to replaced
	}
	if len(addr) == 0 {
		return "", 0, fmt.Errorf("cannot get %s %s child node", c.regionId, c.zooKeeperPath)
	}
	nextUsingIndex := 0
	switch c.balanceMode {
	case LastServer:
		nextUsingIndex = len(addr) - 1
	case RandomServer:
		nextUsingIndex = rand.Intn(len(addr))
	}
	s := &serviceAddress{
		address:        addr,
		nextUsingIndex: nextUsingIndex,
	}
	serviceAddressPool.Store(zKey, s)
	return getAddr(s, c.balanceMode)
}
