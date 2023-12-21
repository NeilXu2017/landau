package data

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"html/template"
	"strconv"
	"strings"
	"sync"
	"time"
)

type (
	ServiceHealthInfo struct {
		ServiceName  string            //service identity
		Address      []string          //service address
		Health       map[string]int    //key: address value: 1 ok, 0 check failure
		CallCount    map[string]uint64 //key: address value: call count tick
		NextSequence int               //next using index of address, when calling,health check again
		AvailableSeq map[int]int       //key: address index value: 1 for ready
		ReceiveTime  map[string]int64  //key: address  value: timer
	}
	_HealthCheckRequest struct {
		Action         string //identity health check request: ServiceHealthCheck
		Service        string //identity checking server
		CheckTime      int64  //request time,unix time
		Checker        string //checking source
		CheckerAddress string //checking source service address
		NotifyShutdown bool   //notify shutdown message, receiver should mark the address unreachable.
	}
	_HealthCheckResponse struct {
		RetCode      int    //response code should be 0
		HealthStatus int    //response health status:  1  ready  0  not ready
		Message      string //response description
	}
	_KeepalivedTraceModel struct {
		QueryTime             string //query timer
		ServiceName           string //identity of this service
		ServiceAddress        string
		ServiceNameHeadTag    string
		ServiceAddressHeadTag string
		HealthCheckPeriod     int
		HealthCheckTimeout    int
		ReceiverKeepTimer     int64
		Node                  []_KeepalivedServiceTraceInfo
		TraceCallerService    []_KeepalivedServiceTraceInfo
		LastTraceService      []_KeepalivedServiceTraceInfo
	}
	_KeepalivedServiceTraceInfo struct {
		ServiceName      string
		AddressNum       int
		FirstAddress     string
		FirstHealth      string
		FirstCallCount   uint64
		FirstReceiveTime string
		OtherAddress     []_KeepalivedServiceRowShowInfo
	}
	_KeepalivedServiceRowShowInfo struct {
		Address     string
		Health      string
		CallCount   uint64
		ReceiveTime string
	}
)

var (
	ServiceName                   = ""                                  //identity of this service
	ServiceAddress                = ""                                  //address of this service
	ServiceNameHeadTag            = "Landau-Service"                    //http head tag name of service id
	ServiceAddressHeadTag         = "Landau-Service-Addr"               //http head name of service address
	serviceHealthMesh             = make(map[string]*ServiceHealthInfo) //key: service name value: health info
	syncServiceMesh               = sync.RWMutex{}                      //sync map variable access
	HealthCheckPeriod             = 5                                   //default period of health checking
	HealthCheckTimeout            = 3                                   //default timeout of health checking
	LastTraceServiceAddress       = sync.Map{}                          //record exited last service address from requester
	receiveServiceMesh            = make(map[string]*ServiceHealthInfo) //receive service address
	syncReceiverService           = sync.RWMutex{}                      //sync map variable receiveServiceMesh
	ReceiverKeepTimer       int64 = 30                                  //keep period to regard as health okay

	//go:embed keepalived_trace.html
	keepalivedTraceFile embed.FS
)

func StartHealthChecking() {
	if len(serviceHealthMesh) > 0 {
		timer := time.NewTicker(time.Duration(HealthCheckPeriod) * time.Second)
		defer timer.Stop()
		for {
			<-timer.C
			_healthChecking(false)
		}
	}
}

func RegisterServiceHealth(serviceName string, serviceAddress []string) {
	serviceHealthMesh[serviceName] = &ServiceHealthInfo{
		ServiceName:  serviceName,
		Address:      serviceAddress,
		Health:       make(map[string]int),
		CallCount:    make(map[string]uint64),
		NextSequence: 0,
		AvailableSeq: make(map[int]int),
		ReceiveTime:  make(map[string]int64),
	}
}

func (c *ServiceHealthInfo) IsExisted(addr string) bool {
	for _, v := range c.Address {
		if v == addr {
			return true
		}
	}
	return false
}

func ChangeServiceHealth(services map[string][]string) {
	syncServiceMesh.Lock()
	defer syncServiceMesh.Unlock()
	for serviceName, serviceAddress := range services {
		if v, ok := serviceHealthMesh[serviceName]; ok {
			for _, addr := range serviceAddress {
				if !v.IsExisted(addr) {
					v.Address = append(v.Address, addr)
				}
			}
		} else {
			serviceHealthMesh[serviceName] = &ServiceHealthInfo{
				ServiceName:  serviceName,
				Address:      serviceAddress,
				Health:       make(map[string]int),
				CallCount:    make(map[string]uint64),
				NextSequence: 0,
				AvailableSeq: make(map[int]int),
				ReceiveTime:  make(map[string]int64),
			}
		}
	}
}

func RemoveReceiveService(serviceName, addr string) {
	syncReceiverService.Lock()
	if c, ok := receiveServiceMesh[serviceName]; ok {
		c.ReceiveTime[addr] = 0
		addrNotFound := true
		for _, d := range c.Address {
			if d == addr {
				addrNotFound = false
				break
			}
		}
		if addrNotFound {
			c.Address = append(c.Address, addr)
		}
	}
	syncReceiverService.Unlock()
}

func RegisterReceiveService(serviceName, addr string) {
	t := time.Now().Unix()
	syncReceiverService.Lock()
	if c, ok := receiveServiceMesh[serviceName]; ok {
		c.ReceiveTime[addr] = t
		addrNotFound := true
		for _, d := range c.Address {
			if d == addr {
				addrNotFound = false
				break
			}
		}
		if addrNotFound {
			c.Address = append(c.Address, addr)
		}
	} else {
		v := &ServiceHealthInfo{
			ServiceName:  serviceName,
			ReceiveTime:  make(map[string]int64),
			Address:      []string{addr},
			CallCount:    make(map[string]uint64),
			Health:       make(map[string]int),
			AvailableSeq: make(map[int]int),
		}
		v.ReceiveTime[addr] = t
		receiveServiceMesh[serviceName] = v
	}
	syncReceiverService.Unlock()
}

func NewServiceHealthCheckRequest() interface{} {
	return &_HealthCheckRequest{}
}

func (c _HealthCheckRequest) String() string {
	b, _ := json.Marshal(c)
	return string(b)
}

func DoHealthCheck(_ *gin.Context, param interface{}) (interface{}, string) {
	req := param.(*_HealthCheckRequest)
	if req.Checker != "" && req.CheckerAddress != "" {
		if req.NotifyShutdown {
			RemoveReceiveService(req.Checker, req.CheckerAddress)
		} else {
			RegisterReceiveService(req.Checker, req.CheckerAddress)
		}
	}
	rsp := _HealthCheckResponse{RetCode: 0, HealthStatus: 1, Message: "HealthCheckResponse"}
	return rsp, req.String()
}

func OutputKeepaliveStatics(c *gin.Context) {
	m := _KeepalivedTraceModel{
		QueryTime:             time.Now().Format("2006-01-02 15:04:05"),
		ServiceName:           ServiceName,
		ServiceAddress:        ServiceAddress,
		ServiceNameHeadTag:    ServiceNameHeadTag,
		ServiceAddressHeadTag: ServiceAddressHeadTag,
		HealthCheckPeriod:     HealthCheckPeriod,
		HealthCheckTimeout:    HealthCheckTimeout,
		ReceiverKeepTimer:     ReceiverKeepTimer,
	}
	syncServiceMesh.RLock()
	defer syncServiceMesh.RUnlock()
	for name, d := range serviceHealthMesh {
		v := _KeepalivedServiceTraceInfo{
			ServiceName:      name,
			AddressNum:       len(d.Address),
			FirstAddress:     d.Address[0],
			FirstHealth:      strconv.Itoa(d.Health[d.Address[0]]),
			FirstCallCount:   d.CallCount[d.Address[0]],
			FirstReceiveTime: time.Unix(d.ReceiveTime[d.Address[0]], 0).Format("2006-01-02 15:04:05"),
		}
		for i := 1; i < len(d.Address); i++ {
			address := d.Address[i]
			o := _KeepalivedServiceRowShowInfo{
				Address:     address,
				Health:      strconv.Itoa(d.Health[address]),
				CallCount:   d.CallCount[address],
				ReceiveTime: time.Unix(d.ReceiveTime[address], 0).Format("2006-01-02 15:04:05"),
			}
			v.OtherAddress = append(v.OtherAddress, o)
		}
		m.Node = append(m.Node, v)
	}
	syncReceiverService.RLock()
	defer syncReceiverService.RUnlock()
	for name, d := range receiveServiceMesh {
		addr := d.Address[0]
		v := _KeepalivedServiceTraceInfo{
			ServiceName:      name,
			FirstAddress:     addr,
			FirstReceiveTime: time.Unix(d.ReceiveTime[addr], 0).Format("2006-01-02 15:04:05"),
			AddressNum:       len(d.Address),
			FirstCallCount:   d.CallCount[addr],
		}
		for i := 1; i < len(d.Address); i++ {
			address := d.Address[i]
			o := _KeepalivedServiceRowShowInfo{
				Address:     address,
				CallCount:   d.CallCount[address],
				ReceiveTime: time.Unix(d.ReceiveTime[address], 0).Format("2006-01-02 15:04:05"),
			}
			v.OtherAddress = append(v.OtherAddress, o)
		}
		m.TraceCallerService = append(m.TraceCallerService, v)
	}
	rangeAdd := func(key, value interface{}) bool {
		m.LastTraceService = append(m.LastTraceService, _KeepalivedServiceTraceInfo{
			ServiceName:  key.(string),
			FirstAddress: value.(string),
		})
		return true
	}
	LastTraceServiceAddress.Range(rangeAdd)
	if t, err := template.ParseFS(keepalivedTraceFile, "keepalived_trace.html"); err == nil {
		var buf bytes.Buffer
		if err := t.Execute(&buf, m); err == nil {
			nbs := buf.Bytes()
			c.Data(200, "text/html", nbs)
			return
		}
	}
	c.Data(200, "text/html", []byte("nothing"))
}

func getServiceAddrByName(serviceName string) string {
	addr, usingSequence := "", 0 //service address
	syncServiceMesh.RLock()
	if v, ok := serviceHealthMesh[serviceName]; ok {
		if len(v.Address) > 0 {
			if len(v.AvailableSeq) == 0 || len(v.Address) == 1 {
				addr = v.Address[0]
			} else {
				tryCount := len(v.Address)
				usingSequence = v.NextSequence
				for {
					if usingSequence >= len(v.Address) {
						usingSequence = 0
					}
					if _, ok := v.AvailableSeq[usingSequence]; ok {
						addr = v.Address[usingSequence]
						break
					}
					usingSequence++
					tryCount--
					if tryCount <= 0 {
						addr, usingSequence = v.Address[0], 0
						break
					}
				}
			}
		}
	}
	syncServiceMesh.RUnlock()
	if addr != "" {
		syncServiceMesh.Lock()
		serviceHealthMesh[serviceName].Increment(addr, usingSequence)
		syncServiceMesh.Unlock()
	}
	if addr == "" {
		usingReceivedSeq := 0
		syncReceiverService.RLock()
		if v, ok := receiveServiceMesh[serviceName]; ok {
			usingReceivedSeq = v.NextSequence
			if len(v.ReceiveTime) > 0 {
				var availableAddr []string
				t := time.Now().Unix()
				for a, c := range v.ReceiveTime {
					if t-c <= ReceiverKeepTimer {
						availableAddr = append(availableAddr, a)
					}
				}
				if len(availableAddr) > 0 {
					if usingReceivedSeq >= len(availableAddr) {
						usingReceivedSeq = 0
					}
					addr = availableAddr[usingReceivedSeq]
				}
			}
		}
		syncReceiverService.RUnlock()
		if addr != "" {
			syncReceiverService.Lock()
			receiveServiceMesh[serviceName].Increment(addr, usingReceivedSeq)
			syncReceiverService.Unlock()
		}
	}
	if addr == "" {
		if v, ok := LastTraceServiceAddress.Load(serviceName); ok {
			addr = v.(string)
		}
	}
	if addr != "" && !strings.Contains(addr, "http://") {
		addr = fmt.Sprintf("http://%s", addr)
	}
	return addr
}

func (c *ServiceHealthInfo) Increment(serviceAddress string, usingSequence int) {
	c.CallCount[serviceAddress] = c.CallCount[serviceAddress] + 1
	c.NextSequence = usingSequence + 1
}

func _updateServerHealthStatus(serviceName, address string, healthStatus int) {
	syncServiceMesh.Lock()
	defer syncServiceMesh.Unlock()
	if v, ok := serviceHealthMesh[serviceName]; ok {
		v.Health[address] = healthStatus
		v.ReceiveTime[address] = time.Now().Unix()
		for index, addr := range v.Address {
			if addr == address {
				if healthStatus == 1 {
					v.AvailableSeq[index] = 1
				} else {
					delete(v.AvailableSeq, index)
				}
				return
			}
		}
	}
}

func _healthChecking(notifyShutdown bool) {
	services := make(map[string]string)
	syncServiceMesh.RLock()
	for name, d := range serviceHealthMesh {
		for _, addr := range d.Address {
			services[addr] = name
		}
	}
	syncServiceMesh.RUnlock()
	wg, checkTime := sync.WaitGroup{}, time.Now().Unix()
	check := func(addr, name string) {
		defer wg.Done()
		req := _HealthCheckRequest{Action: "ServiceHealthCheck", Service: name, CheckTime: checkTime, Checker: ServiceName, CheckerAddress: ServiceAddress, NotifyShutdown: notifyShutdown}
		rsp, healthStatus := &_HealthCheckResponse{}, 0
		httpHelper, _ := NewHTTPHelper(SetHTTPUrl(fmt.Sprintf("%s/ServiceHealthCheck", addr)), SetHTTPTimeout(HealthCheckTimeout), SetHTTPRequestRawObject(req), SetHTTPLogCategory("health_checker"))
		if err := httpHelper.Call2(rsp); err == nil && rsp.RetCode == 0 && rsp.HealthStatus == 1 {
			healthStatus = 1
		}
		_updateServerHealthStatus(name, addr, healthStatus)
	}
	for addr, name := range services {
		wg.Add(1)
		go check(addr, name)
	}
	wg.Wait()
}

func NotifyCheckerShutdown() {
	_healthChecking(true)
}
