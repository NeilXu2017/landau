package zookeeper

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/NeilXu2017/landau/util"

	"github.com/NeilXu2017/landau/log"
	"github.com/samuel/go-zookeeper/zk"
)

type (
	//ZConnector zookeeper 访问对象
	ZConnector struct {
		connStr string
		timeout int
		logger  string
	}
)

const (
	defaultZooKeeperLogger  = "main"
	defaultZooKeeperTimeout = 3
)

var (
	zooKeeperPool sync.Map //conn   *zk.Conn
)

// NewZConnector 构建ZConnect 对象
func NewZConnector(address string, timeout int, loggerName string) *ZConnector {
	z := &ZConnector{
		connStr: address,
		logger:  loggerName,
		timeout: timeout,
	}
	return z
}

// NewZConnector2 构建ZConnect 对象
func NewZConnector2(address string) *ZConnector {
	return NewZConnector(address, defaultZooKeeperTimeout, defaultZooKeeperLogger)
}

// GetConn 返回 zk.Conn 对象
func (c *ZConnector) GetConn() (*zk.Conn, error) {
	if conn, ok := zooKeeperPool.Load(c.connStr); ok {
		zConn := conn.(*zk.Conn)
		if zConn.State() == zk.StateHasSession {
			return zConn, nil
		}
	}
	zAddr := util.SplitAddrConvert(c.connStr, util.IPV6Bracket)
	zServerAddr := strings.Split(zAddr, ",")
	zConn, ec, err := zk.Connect(zServerAddr, time.Duration(c.timeout)*time.Second, zk.WithLogInfo(false))
	if err == nil {
		for {
			select {
			case connEvent, ok := <-ec:
				if ok {
					switch connEvent.State {
					case zk.StateHasSession:
						zooKeeperPool.Store(c.connStr, zConn)
						return zConn, nil
					default:
						continue
					}
				} else {
					return nil, fmt.Errorf("[Zookeeper] Connect address %s error", zServerAddr)
				}
			default:
				continue
			}
		}
	}
	return zConn, err
}

// GetNode 获取节点数据
func (c *ZConnector) GetNode(path string) ([]byte, *zk.Stat, error) {
	zc, err := c.GetConn()
	if err == nil {
		return zc.Get(path)
	}
	return nil, nil, err
}

func (c *ZConnector) convertPath(path string) (string, []string) {
	originPaths := strings.Split(path, "/")
	var paths []string
	for _, p := range originPaths {
		if p != "" {
			paths = append(paths, p)
		}
	}
	return "", paths
}

// CreateNode 创建节点
func (c *ZConnector) CreateNode(path string, data []byte) (string, error) {
	start := time.Now()
	var createdPath string
	var createdError error
	defer func() {
		if createdError != nil {
			log.Error2(c.logger, "[Zookeeper]\t[%s]\tCreateNode created path:%s data len=%d error:%v", time.Since(start), createdPath, len(data), createdError)
		} else {
			log.Info2(c.logger, "[Zookeeper]\t[%s]\tCreateNode created path:%s data len=%d", time.Since(start), createdPath, len(data))
		}
	}()

	if path == "" {
		createdError = fmt.Errorf("empty path")
		return "", createdError
	}
	acl := zk.WorldACL(zk.PermAll)
	zc, err := c.GetConn()
	if err != nil {
		createdError = err
		return "", createdError
	}
	currentNodePath, paths := c.convertPath(path)
	for i, j := 0, len(paths); i < j; i++ {
		currentNodePath += "/" + paths[i]
		exist, stat, err := zc.Exists(currentNodePath)
		if err != nil {
			return "", err
		}
		isLastNode := i == j-1
		if exist {
			if isLastNode {
				_, createdError = zc.Set(currentNodePath, data, stat.Version)
			}
		} else {
			var nodeData []byte
			var nodeFlag int32
			if isLastNode {
				nodeFlag = int32(zk.FlagEphemeral)
				nodeData = data
			}
			createdPath, createdError = zc.Create(currentNodePath, nodeData, nodeFlag, acl)
		}
	}
	return createdPath, createdError
}

// SetNode 设置节点数据
func (c *ZConnector) SetNode(path string, data []byte) error {
	start := time.Now()
	var setError error
	defer func() {
		if setError != nil {
			log.Error2(c.logger, "[Zookeeper]\t[%s]\tSetNode path:%s data len=%d error:%v", time.Since(start), path, len(data), setError)
		} else {
			log.Info2(c.logger, "[Zookeeper]\t[%s]\tSetNode path:%s data len=%d", time.Since(start), path, len(data))
		}
	}()
	if path == "" {
		setError = fmt.Errorf("empty path")
		return setError
	}
	zc, err := c.GetConn()
	if err != nil {
		setError = err
		return setError
	}
	exist, stat, err := zc.Exists(path)
	if err != nil {
		setError = err
		return err
	}
	if !exist {
		setError = fmt.Errorf("node path:%s not exist", path)
		return setError
	}
	_, setError = zc.Set(path, data, stat.Version)
	return setError
}

// DeleteNode 删除节点
func (c *ZConnector) DeleteNode(path string) error {
	zc, err := c.GetConn()
	if err != nil {
		return err
	}
	exist, stat, err := zc.Exists(path)
	if err != nil {
		return err
	}
	if exist {
		return zc.Delete(path, stat.Version)
	}
	return nil
}
