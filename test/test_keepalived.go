package test

import (
	"encoding/json"
	"fmt"
	"github.com/NeilXu2017/landau/api"
	"github.com/NeilXu2017/landau/data"
	"github.com/NeilXu2017/landau/entry"
	"github.com/NeilXu2017/landau/log"
	"github.com/gin-gonic/gin"
	"strings"
)

type (
	callServiceNameRequest struct {
		Action string
		Name   string //调用的service name
		Caller string //调用
	}
	callServiceNameResponse struct {
		Code            int
		Message         string
		ServiceResponse interface{}
	}

	testCheckKeepaliveRequest struct {
		Action string
		Caller string
	}
	testCheckKeepaliveResponse struct {
		Code                int
		ResponseServiceName string
	}
)

func newTestCheckKeepaliveRequest() interface{} {
	return &testCheckKeepaliveRequest{}
}
func (c *testCheckKeepaliveRequest) String() string {
	b, _ := json.Marshal(c)
	return string(b)
}
func doTestCheckKeepalive(c *gin.Context, param interface{}) (interface{}, string) {
	reqeust := param.(*testCheckKeepaliveRequest)
	rsp := testCheckKeepaliveResponse{
		Code:                0,
		ResponseServiceName: fmt.Sprintf("Hi %s,Response from service:%s", reqeust.Caller, data.ServiceName),
	}
	return rsp, reqeust.String()
}
func (c *callServiceNameRequest) String() string {
	b, _ := json.Marshal(c)
	return string(b)
}
func newCallServiceNameRequest() interface{} {
	return &callServiceNameRequest{}
}

func doCallServiceName(c *gin.Context, param interface{}) (interface{}, string) {
	reqeust := param.(*callServiceNameRequest)
	result := callServiceNameResponse{}
	if reqeust.Name != "" {
		req := testCheckKeepaliveRequest{
			Action: "CallServiceByServiceName",
			Caller: fmt.Sprintf("%s_%s", data.ServiceName, reqeust.Caller),
		}
		rsp := testCheckKeepaliveResponse{}
		data.CallHTTPServiceEx2(reqeust.Name, req, &rsp)
		result.ServiceResponse = rsp
	}
	return result, reqeust.String()
}

func doStarting(c *gin.Context, param interface{}) (interface{}, string) {
	reqeust := param.(*callServiceNameRequest)
	result := callServiceNameResponse{}
	api.SetServiceReady(true)
	return result, reqeust.String()
}

func KeepalivedClient(httpPort int) {
	s := &entry.LandauServer{
		LogConfig:          logConfigContent,
		DefaultLoggerName:  entry.DefaultLogger,
		GinLoggerName:      entry.DefaultGinLogger,
		HTTPServiceAddress: "127.0.0.1",
		HTTPServicePort:    httpPort,
		GRPCServicePort:    0,
		RegisterGRPCHandle: registerRGPCHandle,
		CustomInit:         myCustomInit,
		//GetCronTasks:              getCronTasks,
		RegisterHTTPHandles:       registerHTTPHandles,
		RegisterHTTPCustomHandles: registerHTTPCustomHandles,
		HTTPNeedCheckACL:          true,
		HTTPCheckACL:              myCheckACL,
		HTTPEnableCustomLogTag:    true,
		HTTPCustomLog:             getUserSessionID,
		EnablePrometheusMetric:    true,
		PrometheusMetricNamespace: "landau",
		ServiceName:               "HostAgent",
		ServiceAddress:            fmt.Sprintf("http://127.0.0.1:%d", httpPort),
		DynamicReloadConfig: func() {
			log.Info("[DynamicReloadConfig] received, changing app biz config.")
		},
	}
	s.Start()
}

func KeepalivedClient2(httpPort int, primaryIp, secondaryIp string) {
	s := &entry.LandauServer{
		LogConfig:                 logConfigContent,
		DefaultLoggerName:         entry.DefaultLogger,
		GinLoggerName:             entry.DefaultGinLogger,
		HTTPServiceAddress:        primaryIp,
		SecondaryServiceAddress:   secondaryIp,
		HTTPServicePort:           httpPort,
		GRPCServicePort:           0,
		RegisterGRPCHandle:        registerRGPCHandle,
		CustomInit:                myCustomInit,
		RegisterHTTPHandles:       registerHTTPHandles,
		RegisterHTTPCustomHandles: registerHTTPCustomHandles,
		HTTPNeedCheckACL:          true,
		HTTPCheckACL:              myCheckACL,
		HTTPEnableCustomLogTag:    true,
		HTTPCustomLog:             getUserSessionID,
		EnablePrometheusMetric:    false,
		PrometheusMetricNamespace: "landau",
		ServiceName:               "HostAgent",
		ServiceAddress:            fmt.Sprintf("http://%s:%d", primaryIp, httpPort),
		DynamicReloadConfig: func() {
			log.Info("[DynamicReloadConfig] received, changing app biz config.")
		},
	}
	s.Start()
}

func KeepalivedService(serviceName string, httpPort int, checkServiceName string) {
	getCheckServiceHealth := func() map[string][]string {
		m := make(map[string][]string)
		services := strings.Split(checkServiceName, "$")
		for _, s := range services {
			list := strings.Split(s, ",")
			if len(list) > 1 {
				var address []string
				for i := 1; i < len(list); i++ {
					address = append(address, list[i])
				}
				m[list[0]] = address
			}
		}
		return m
	}
	s := &entry.LandauServer{
		LogConfig:                 logConfigContent,
		DefaultLoggerName:         entry.DefaultLogger,
		GinLoggerName:             entry.DefaultGinLogger,
		HTTPServiceAddress:        "127.0.0.1",
		HTTPServicePort:           httpPort,
		GRPCServicePort:           0,
		RegisterGRPCHandle:        registerRGPCHandle,
		CustomInit:                myCustomInit,
		GetCronTasks:              getCronTasks,
		RegisterHTTPHandles:       registerHTTPHandles,
		RegisterHTTPCustomHandles: registerHTTPCustomHandles,
		HTTPNeedCheckACL:          true,
		HTTPCheckACL:              myCheckACL,
		HTTPEnableCustomLogTag:    true,
		HTTPCustomLog:             getUserSessionID,
		EnablePrometheusMetric:    true,
		PrometheusMetricNamespace: serviceName,
		ServiceName:               serviceName,
		ServiceAddress:            fmt.Sprintf("http://127.0.0.1:%d", httpPort),
		CheckServiceHealth:        getCheckServiceHealth,
		DynamicReloadConfig: func() {
			log.Info("[DynamicReloadConfig] received, changing app biz config.")
		},
	}
	s.Start()
}

func KeepalivedService2(serviceName string, httpPort int, primaryIp, secondaryIp string, checkServiceName string) {
	//需要进行健康检查的 service,service 有第2个地址
	getCheckServiceHealth := func() (map[string][]string, map[string]string) {
		m, n := make(map[string][]string), make(map[string]string)
		services := strings.Split(checkServiceName, "$")
		for _, s := range services {
			ss := strings.Split(s, "^")
			list := strings.Split(ss[0], ",")
			if len(list) > 1 {
				var address []string
				for i := 1; i < len(list); i++ {
					address = append(address, list[i])
				}
				m[list[0]] = address
				if len(ss) > 1 {
					pairs := strings.Split(ss[1], ",")
					for _, pair := range pairs {
						if v := strings.Split(pair, "#"); len(v) == 2 {
							n[v[0]] = v[1]
						}
					}
				}
			}
		}
		return m, n
	}

	s := &entry.LandauServer{
		LogConfig:                 logConfigContent,
		DefaultLoggerName:         entry.DefaultLogger,
		GinLoggerName:             entry.DefaultGinLogger,
		HTTPServiceAddress:        primaryIp,
		SecondaryServiceAddress:   secondaryIp,
		HTTPServicePort:           httpPort,
		GRPCServicePort:           0,
		RegisterGRPCHandle:        registerRGPCHandle,
		CustomInit:                myCustomInit,
		RegisterHTTPHandles:       registerHTTPHandles,
		RegisterHTTPCustomHandles: registerHTTPCustomHandles,
		HTTPNeedCheckACL:          true,
		HTTPCheckACL:              myCheckACL,
		HTTPEnableCustomLogTag:    true,
		HTTPCustomLog:             getUserSessionID,
		EnablePrometheusMetric:    true,
		PrometheusMetricNamespace: serviceName,
		ServiceName:               serviceName,
		ServiceAddress:            fmt.Sprintf("http://%s:%d", primaryIp, httpPort),
		CheckServiceHealth2:       getCheckServiceHealth,
		DynamicReloadConfig: func() {
			log.Info("[DynamicReloadConfig] received, changing app biz config.")
		},
	}
	s.Start()
}
