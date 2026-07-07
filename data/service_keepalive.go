package data

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"github.com/NeilXu2017/landau/log"
	"github.com/gin-gonic/gin"
	"sort"
	"strconv"
	"strings"
	"sync"
	"text/template"
	"time"
)

type (
	ServiceHealthInfo struct {
		ServiceName       string            //service identity
		Address           []string          //service address
		Health            map[string]int    //key: address value: 1 ok, 0 check failure
		HealthOnSecondary map[string]int    //health =1 using secondary address
		CallCount         map[string]uint64 //key: address value: call count tick
		NextSequence      int               //next using index of address, when calling,health check again
		AvailableSeq      map[int]int       //key: address index value: 1 for ready
		ReceiveTime       map[string]int64  //key: address  value: timer
	}
	_HealthCheckRequest struct {
		Action           string //identity health check request: ServiceHealthCheck
		Service          string //identity checking server
		CheckTime        int64  //request time,unix time
		Checker          string //checking source
		CheckerAddress   string //checking source service address
		PrimaryAddress   string //service primary address
		SecondaryAddress string //service secondary address
		NotifyShutdown   bool   //notify shutdown message, receiver should mark the address unreachable.
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
	_SortKeepalivedServiceTraceInfo []_KeepalivedServiceTraceInfo
)

var (
	ServiceName                                                                   = ""                                  //identity of this service
	ServiceAddress                                                                = ""                                  //address of this service
	SecondaryServiceAddress                                                       = ""                                  //secondary address of this service
	ServiceNameHeadTag                                                            = "Landau-Service"                    //http head tag name of service id
	ServiceAddressHeadTag                                                         = "Landau-Service-Addr"               //http head name of service address
	serviceHealthMesh                                                             = make(map[string]*ServiceHealthInfo) //key: service name value: health info
	syncServiceMesh                                                               = sync.RWMutex{}                      //sync map variable access
	HealthCheckPeriod                                                             = 5                                   //default period of health checking
	HealthCheckTimeout                                                            = 3                                   //default timeout of health checking
	LastTraceServiceAddress                                                       = sync.Map{}                          //record exited last service address from requester
	receiveServiceMesh                                                            = make(map[string]*ServiceHealthInfo) //receive service address
	syncReceiverService                                                           = sync.RWMutex{}                      //sync map variable receiveServiceMesh
	ReceiverKeepTimer             int64                                           = 30                                  //keep period to regard as health okay
	MonitorServiceAddrChange      func() map[string][]string                                                            //monitor the config service address changed
	MonitorServiceAddrChange2     func() (map[string][]string, map[string]string)                                       //monitor the config service address changed, secondary service map
	MonitorServiceAddrPeriod      = 15                                                                                  //monitor job period
	lastServiceAddr               map[string][]string                                                                   //last service address
	ServiceMeshPrimary2Secondary  = make(map[string]string)                                                             //other service secondary address key: primary address  value: secondary address
	ServiceMeshSecondary2Primary  = make(map[string]string)                                                             //other service secondary address key:secondary address value: primary address
	syncMeshPrimary               = sync.RWMutex{}                                                                      //sync map
	DisableAssignSourceIp         bool                                                                                  //disable source ip
	lastReceiveServerCheckAddress = make(map[string]string)                                                             //last received server checker
	syncLastServerChecker         = sync.RWMutex{}                                                                      //sync lastReceiveServerCheckAddress
	serviceCallback               = sync.Map{}                                                                          //record callback zec
	ReceivedServiceCallback       func(string, string) bool                                                             //收到服务推送地址 回调设置

	//go:embed keepalived_trace.html
	keepalivedTraceFile embed.FS
)

func (a _SortKeepalivedServiceTraceInfo) Len() int      { return len(a) }
func (a _SortKeepalivedServiceTraceInfo) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a _SortKeepalivedServiceTraceInfo) Less(i, j int) bool {
	if a[i].ServiceName == a[j].ServiceName {
		return a[i].AddressNum < a[j].AddressNum
	}
	return a[i].ServiceName < a[j].ServiceName
}

func MonitorServiceHealthConfigs() {
	if MonitorServiceAddrChange != nil || MonitorServiceAddrChange2 != nil {
		timer := time.NewTicker(time.Duration(MonitorServiceAddrPeriod) * time.Second)
		defer timer.Stop()
		for {
			<-timer.C
			_checkServiceHealthConfig()
		}
	}
}

func isServiceHealthConfigChanged(m map[string][]string) bool {
	if lastServiceAddr == nil || len(lastServiceAddr) != len(m) {
		return true
	}
	p := make(map[string]string)
	for k, v := range lastServiceAddr {
		sort.Strings(v)
		p[k] = strings.Join(v, ",")
		if _, ok := m[k]; !ok {
			return true
		}
	}
	for k, v := range m {
		sort.Strings(v)
		pv := p[k]
		if pv != strings.Join(v, ",") {
			return true
		}
	}
	return false
}

func _checkServiceHealthConfig() {
	if MonitorServiceAddrChange2 != nil {
		m, secondary := MonitorServiceAddrChange2()
		syncMeshPrimary.Lock()
		ServiceMeshPrimary2Secondary = secondary
		ServiceMeshSecondary2Primary = make(map[string]string, 0)
		for k, v := range secondary {
			ServiceMeshSecondary2Primary[v] = k
		}
		syncMeshPrimary.Unlock()
		if isServiceHealthConfigChanged(m) {
			ChangeServiceHealth(m)
			lastServiceAddr = m
		}
	} else {
		m := MonitorServiceAddrChange()
		if isServiceHealthConfigChanged(m) {
			ChangeServiceHealth(m)
			lastServiceAddr = m
		}
	}
}

func StartHealthChecking() {
	timer := time.NewTicker(time.Duration(HealthCheckPeriod) * time.Second)
	defer timer.Stop()
	for {
		if len(serviceHealthMesh) > 0 {
			_healthChecking(false)
		}
		<-timer.C
	}
}

func RegisterServiceHealth() {
	if MonitorServiceAddrChange2 != nil {
		syncMeshPrimary.Lock()
		lastServiceAddr, ServiceMeshPrimary2Secondary = MonitorServiceAddrChange2()
		ServiceMeshSecondary2Primary = make(map[string]string, 0)
		for k, v := range ServiceMeshPrimary2Secondary {
			ServiceMeshSecondary2Primary[v] = k
		}
		syncMeshPrimary.Unlock()
	} else {
		lastServiceAddr = MonitorServiceAddrChange()
	}
	for serviceName, address := range lastServiceAddr {
		serviceHealthMesh[serviceName] = &ServiceHealthInfo{
			ServiceName:       serviceName,
			Address:           address,
			Health:            make(map[string]int),
			HealthOnSecondary: make(map[string]int),
			CallCount:         make(map[string]uint64),
			NextSequence:      0,
			AvailableSeq:      make(map[int]int),
			ReceiveTime:       make(map[string]int64),
		}
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
				ServiceName:       serviceName,
				Address:           serviceAddress,
				Health:            make(map[string]int),
				HealthOnSecondary: make(map[string]int),
				CallCount:         make(map[string]uint64),
				NextSequence:      0,
				AvailableSeq:      make(map[int]int),
				ReceiveTime:       make(map[string]int64),
			}
		}
	}
}

func RemoveReceiveService(serviceName, addr string) {
	syncReceiverService.Lock()
	primaryAddr := addr
	syncMeshPrimary.RLock()
	if v, ok := ServiceMeshSecondary2Primary[addr]; ok {
		primaryAddr = v
	}
	syncMeshPrimary.RUnlock()
	if c, ok := receiveServiceMesh[serviceName]; ok {
		c.ReceiveTime[primaryAddr] = 0
		addrNotFound := true
		for _, d := range c.Address {
			if d == addr {
				addrNotFound = false
				break
			}
		}
		if addrNotFound {
			c.Address = append(c.Address, primaryAddr)
		}
	}
	syncReceiverService.Unlock()
}

func RegisterReceiveService(serviceName, addr string) {
	t := time.Now().Unix()
	syncReceiverService.Lock()
	primaryAddr := addr
	syncMeshPrimary.RLock()
	if v, ok := ServiceMeshSecondary2Primary[addr]; ok {
		primaryAddr = v
	}
	syncMeshPrimary.RUnlock()
	if c, ok := receiveServiceMesh[serviceName]; ok {
		c.ReceiveTime[primaryAddr] = t
		if primaryAddr != addr {
			c.HealthOnSecondary[primaryAddr] = 1
		} else {
			delete(c.HealthOnSecondary, primaryAddr)
		}
		addrNotFound := true
		for _, d := range c.Address {
			if d == primaryAddr {
				addrNotFound = false
				break
			}
		}
		if addrNotFound {
			c.Address = append(c.Address, primaryAddr)
		}
	} else {
		v := &ServiceHealthInfo{
			ServiceName:       serviceName,
			ReceiveTime:       make(map[string]int64),
			Address:           []string{primaryAddr},
			CallCount:         make(map[string]uint64),
			Health:            make(map[string]int),
			HealthOnSecondary: make(map[string]int),
			AvailableSeq:      make(map[int]int),
		}
		v.ReceiveTime[primaryAddr] = t
		if primaryAddr != addr {
			v.HealthOnSecondary[addr] = 1
		}
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
		if req.PrimaryAddress != "" && req.SecondaryAddress != "" && req.PrimaryAddress != req.SecondaryAddress {
			httpPrimary := fmt.Sprintf(`http://%s`, req.PrimaryAddress)
			httpSecondary := fmt.Sprintf(`http://%s`, req.SecondaryAddress)
			syncMeshPrimary.Lock()
			ServiceMeshPrimary2Secondary[req.PrimaryAddress] = req.SecondaryAddress
			ServiceMeshSecondary2Primary[req.SecondaryAddress] = req.PrimaryAddress
			ServiceMeshPrimary2Secondary[httpPrimary] = httpSecondary
			ServiceMeshSecondary2Primary[httpSecondary] = httpPrimary
			syncMeshPrimary.Unlock()
		}
		key, lastCheckerAddress := fmt.Sprintf("%s_%s", req.Checker, req.PrimaryAddress), ""
		syncLastServerChecker.RLock()
		lastCheckerAddress = lastReceiveServerCheckAddress[key]
		syncLastServerChecker.RUnlock()
		if lastCheckerAddress != req.CheckerAddress {
			if req.CheckerAddress == req.PrimaryAddress {
				log.Info("[%s]-[PrimaryAddress:%s]-[SecondaryAddress:%s] using primary address now", req.Checker, req.PrimaryAddress, req.SecondaryAddress)
			} else {
				log.Info("[%s]-[PrimaryAddress:%s]-[SecondaryAddress:%s] using secondary address now", req.Checker, req.PrimaryAddress, req.SecondaryAddress)
			}
			syncLastServerChecker.Lock()
			lastReceiveServerCheckAddress[key] = req.CheckerAddress
			syncLastServerChecker.Unlock()
		}
		if req.NotifyShutdown {
			RemoveReceiveService(req.Checker, req.CheckerAddress)
		} else {
			RegisterReceiveService(req.Checker, req.CheckerAddress)
		}
		if ReceivedServiceCallback != nil && req.Checker != "" && !req.NotifyShutdown {
			if _, ok := serviceCallback.Load(req.Checker); !ok {
				go func() {
					if ReceivedServiceCallback(req.Checker, req.PrimaryAddress) {
						serviceCallback.Store(req.Checker, struct{}{})
					}
				}()
			}
		}
	}
	rsp := _HealthCheckResponse{RetCode: 0, HealthStatus: 1, Message: "HealthCheckResponse"}
	return rsp, req.String()
}

func outputServiceAddress(addr string) (string, bool) {
	outputAddr, haveSecondaryAddr := addr, false
	syncMeshPrimary.RLock()
	if v, ok := ServiceMeshPrimary2Secondary[addr]; ok {
		outputAddr = fmt.Sprintf(`%s <label style="color:red">%s</label>`, addr, v)
		haveSecondaryAddr = true
	}
	syncMeshPrimary.RUnlock()
	return outputAddr, haveSecondaryAddr
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
	if SecondaryServiceAddress != "" {
		m.ServiceAddress = fmt.Sprintf(`%s <label style="color:red">%s</label>`, ServiceAddress, SecondaryServiceAddress)
	}
	syncServiceMesh.RLock()
	defer syncServiceMesh.RUnlock()
	for name, d := range serviceHealthMesh {
		addr, haveSecondaryAddr := outputServiceAddress(d.Address[0])
		h := strconv.Itoa(d.Health[d.Address[0]])
		if haveSecondaryAddr {
			if _, ok := d.HealthOnSecondary[d.Address[0]]; ok {
				if d.Health[d.Address[0]] == 1 {
					h = `<label style="color:red">1</label>`
				}
			}
		}
		v := _KeepalivedServiceTraceInfo{
			ServiceName:      name,
			AddressNum:       len(d.Address),
			FirstAddress:     addr,
			FirstHealth:      h,
			FirstCallCount:   d.CallCount[d.Address[0]],
			FirstReceiveTime: time.Unix(d.ReceiveTime[d.Address[0]], 0).Format("2006-01-02 15:04:05"),
		}
		for i := 1; i < len(d.Address); i++ {
			address := d.Address[i]
			addr, haveSecondaryAddr := outputServiceAddress(address)
			h := strconv.Itoa(d.Health[address])
			if haveSecondaryAddr {
				if _, ok := d.HealthOnSecondary[d.Address[i]]; ok {
					if d.Health[d.Address[i]] == 1 {
						h = `<label style="color:red">1</label>`
					}
				}
			}
			o := _KeepalivedServiceRowShowInfo{
				Address:     addr,
				Health:      h,
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
	if len(m.Node) > 0 {
		sort.Sort(_SortKeepalivedServiceTraceInfo(m.Node))
	}
	if len(m.TraceCallerService) > 0 {
		sort.Sort(_SortKeepalivedServiceTraceInfo(m.TraceCallerService))
	}
	if len(m.LastTraceService) > 0 {
		sort.Sort(_SortKeepalivedServiceTraceInfo(m.LastTraceService))
	}
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

func GetServiceAddrByName(serviceName string) (string, bool) {
	addr, usingSequence, isPrimary := "", 0, true //service address
	syncServiceMesh.RLock()
	if v, ok := serviceHealthMesh[serviceName]; ok {
		if len(v.Address) > 0 {
			if len(v.AvailableSeq) == 0 || len(v.Address) == 1 {
				addr = v.Address[0]
				if _, ok := v.HealthOnSecondary[addr]; ok {
					syncMeshPrimary.RLock()
					if secondaryAddr := ServiceMeshPrimary2Secondary[addr]; secondaryAddr != "" {
						addr = secondaryAddr
						isPrimary = false
					}
					syncMeshPrimary.RUnlock()
				}
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
				if _, ok := v.HealthOnSecondary[addr]; ok {
					syncMeshPrimary.RLock()
					if secondaryAddr := ServiceMeshPrimary2Secondary[addr]; secondaryAddr != "" {
						addr = secondaryAddr
						isPrimary = false
					}
					syncMeshPrimary.RUnlock()
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
			if _, ok := v.HealthOnSecondary[addr]; ok {
				syncMeshPrimary.RLock()
				if secondaryAddr := ServiceMeshPrimary2Secondary[addr]; secondaryAddr != "" {
					addr = secondaryAddr
					isPrimary = false
				}
				syncMeshPrimary.RUnlock()
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
	return addr, isPrimary
}

func (c *ServiceHealthInfo) Increment(serviceAddress string, usingSequence int) {
	c.CallCount[serviceAddress] = c.CallCount[serviceAddress] + 1
	c.NextSequence = usingSequence + 1
}

func _updateServerHealthStatus(serviceName, address string, healthStatus int) {
	syncServiceMesh.Lock()
	defer syncServiceMesh.Unlock()
	if v, ok := serviceHealthMesh[serviceName]; ok {
		primaryAddr := address
		syncMeshPrimary.RLock()
		if v, ok := ServiceMeshSecondary2Primary[address]; ok {
			primaryAddr = v
		}
		syncMeshPrimary.RUnlock()
		v.Health[primaryAddr] = healthStatus
		v.ReceiveTime[primaryAddr] = time.Now().Unix()
		if primaryAddr == address {
			if healthStatus == 1 {
				delete(v.HealthOnSecondary, address)
			}
		} else {
			v.HealthOnSecondary[address] = 1
		}
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
		req := _HealthCheckRequest{Action: "ServiceHealthCheck", Service: name, CheckTime: checkTime, Checker: ServiceName, CheckerAddress: ServiceAddress, NotifyShutdown: notifyShutdown, PrimaryAddress: ServiceAddress, SecondaryAddress: SecondaryServiceAddress}
		rsp, healthStatus := &_HealthCheckResponse{}, 0
		httpHelper, _ := NewHTTPHelper(SetHTTPUrl(fmt.Sprintf("%s/ServiceHealthCheck", addr)), SetHTTPTimeout(HealthCheckTimeout),
			SetHTTPRequestRawObject(req), SetHTTPLogCategory("health_checker"), SetHTTPDisableAssignSourceIp(DisableAssignSourceIp), SetHTTPIsPrimaryAddress(true))
		if err := httpHelper.Call2(rsp); err == nil && rsp.RetCode == 0 && rsp.HealthStatus == 1 {
			healthStatus = 1
		}
		_updateServerHealthStatus(name, addr, healthStatus)
		if healthStatus == 0 { //check secondary address
			secondaryAddress := ""
			syncMeshPrimary.RLock()
			if v, ok := ServiceMeshPrimary2Secondary[addr]; ok {
				secondaryAddress = v
			}
			syncMeshPrimary.RUnlock()
			if secondaryAddress != "" {
				secReq := _HealthCheckRequest{Action: "ServiceHealthCheck", Service: name, CheckTime: checkTime, Checker: ServiceName, CheckerAddress: SecondaryServiceAddress, NotifyShutdown: notifyShutdown, PrimaryAddress: ServiceAddress, SecondaryAddress: SecondaryServiceAddress}
				secHttpHelper, _ := NewHTTPHelper(SetHTTPUrl(fmt.Sprintf("%s/ServiceHealthCheck", secondaryAddress)), SetHTTPTimeout(HealthCheckTimeout),
					SetHTTPRequestRawObject(secReq), SetHTTPLogCategory("health_checker"), SetHTTPDisableAssignSourceIp(DisableAssignSourceIp), SetHTTPIsPrimaryAddress(false))
				if err := secHttpHelper.Call2(rsp); err == nil && rsp.RetCode == 0 && rsp.HealthStatus == 1 {
					_updateServerHealthStatus(name, secondaryAddress, 1)
				}
			}
		}
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
