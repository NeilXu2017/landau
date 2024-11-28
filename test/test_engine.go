package test

import (
	"encoding/json"
	"time"

	"google.golang.org/grpc"

	"github.com/NeilXu2017/landau/log"
	"github.com/NeilXu2017/landau/util"

	"github.com/NeilXu2017/landau/api"
	"github.com/NeilXu2017/landau/entry"
	"github.com/gin-gonic/gin"
)

type (
	myCronProxy struct{}

	oneInstance          struct{ Name string }
	ConsoleErrorResponse struct {
		HttpCode int    `json:"-"`       //重设 HttpCode
		Code     string `json:"code"`    //错误代码
		Message  string `json:"message"` //错误描述
	}
)

var (
	startTicket = 0
	BadRequest  = ConsoleErrorResponse{HttpCode: 400, Code: "BadRequest", Message: "Bad request parameters or illegal request."}
)

func (c *myCronProxy) Heart() {
	startTicket++
	log.Info("[myCron] heart:%d", startTicket)
	CheckLockServer()
}

func (c *myCronProxy) Scan() {
	log.Info("[myCron] scan called start...")
	time.Sleep(5 * time.Second)
	log.Info("[myCron] scan called sleep(5) down...end")
}

func (c *myCronProxy) Heart2() {
	log.Info("[myCron] Heart2 called")
}

func (c *oneInstance) Hello() {
	log.Info("[oneInstance] [%s] receiver is pointer Hello world %d", c.Name, time.Now().Unix())
	time.Sleep(5 * time.Second)
}

func (c *oneInstance) Hello2() {
	log.Info("[oneInstance] [%s] receiver is no-pointer  Hello2 world %d", c.Name, time.Now().Unix())
}

// CheckEngine 测试Engine
func CheckEngine() {
	s := &entry.LandauServer{
		LogConfig:                 logConfigContent,
		DefaultLoggerName:         entry.DefaultLogger,
		GinLoggerName:             entry.DefaultGinLogger,
		HTTPServiceAddress:        "127.0.0.1",
		HTTPServicePort:           9080,
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
		PrometheusMetricNamespace: "landau",
		DynamicReloadConfig: func() {
			log.Info("[DynamicReloadConfig] received, changing app biz config.")
		},
	}
	s.Start()
}

func CheckCronJobMode() {
	s := &entry.LandauServer{
		LogConfig:          logConfigContent,
		DefaultLoggerName:  entry.DefaultLogger,
		GinLoggerName:      entry.DefaultGinLogger,
		HTTPServiceAddress: "",
		HTTPServicePort:    0,
		GRPCServicePort:    0,
		CustomInit:         myCustomInit,
		GetCronTasks:       getCronTasks,
		DynamicReloadConfig: func() {
			log.Info("[DynamicReloadConfig] received, changing app biz config.")
		},
	}
	s.StartCronJobMode(60)
}

func CheckNormalServerMode() {
	s := &entry.LandauServer{
		LogConfig:          logConfigContent,
		DefaultLoggerName:  entry.DefaultLogger,
		GinLoggerName:      entry.DefaultGinLogger,
		HTTPServiceAddress: "",
		HTTPServicePort:    0,
		GRPCServicePort:    0,
		CustomInit:         myCustomInit,
		GetCronTasks:       getCronTasks,
		DynamicReloadConfig: func() {
			log.Info("[DynamicReloadConfig] received, changing app biz config.")
		},
	}
	s.StartNormalServerMode(func() {
		for {
			time.Sleep(3 * time.Second)
			log.Info("NormalServer do something...")
		}
	}, 60)
}

func CheckNormalMode() {
	s := &entry.LandauServer{
		LogConfig:         logConfigContent,
		DefaultLoggerName: entry.DefaultLogger,
		CustomInit:        myCustomInit,
		GetCronTasks:      getCronTasks,
	}
	s.StartNormalMode(func() {
		for i, j := 0, 5; i < j; i++ {
			time.Sleep(3 * time.Second)
			log.Info("NormalMode do something...")
		}
	})
}

func myCustomInit() {
	log.Info("my custom init...")
}

func getCronTasks() (interface{}, []util.SingletonCronTask) {
	s := &myCronProxy{}
	var cronJobs []util.SingletonCronTask
	c := util.SingletonCronTask{
		Name:      "heart",
		Enable:    true,
		Schedule:  "@every 22s",
		FuncName:  "Heart",
		Immediate: true,
	}
	cronJobs = append(cronJobs, c)
	skipJob := util.SingletonCronTask{
		Name:      "scan",
		Enable:    true,
		Schedule:  "@every 10s",
		FuncName:  "Scan",
		Immediate: true,
	}
	cronJobs = append(cronJobs, skipJob)
	echo := util.SingletonCronTask{
		Name:         "EchoTime",
		Enable:       true,
		Schedule:     "@every 7s",
		FuncName:     "",
		CallbackFunc: echoTime,
		Immediate:    true,
	}
	cronJobs = append(cronJobs, echo)
	hello := util.SingletonCronTask{
		Name:     "Hello",
		Enable:   true,
		Schedule: "@every 3s",
		FuncName: "Hello",
		Instance: &oneInstance{Name: "One"},
	}
	cronJobs = append(cronJobs, hello)

	hello2 := util.SingletonCronTask{
		Name:     "Hello2",
		Enable:   true,
		Schedule: "@every 9s",
		FuncName: "Hello2",
		Instance: &oneInstance{Name: "Two"},
	}
	cronJobs = append(cronJobs, hello2)
	Heart2 := util.SingletonCronTask{
		Name:     "Heart2",
		Enable:   true,
		Schedule: "@every 6s",
		FuncName: "Heart2",
	}
	cronJobs = append(cronJobs, Heart2)
	weekend := util.SingletonCronTask{
		Name:     "WeekEnd",
		Enable:   true,
		Schedule: "@every 4s",
		FuncName: "WeekEnd",
		Instance: &oneInstance{Name: "Three"},
	}
	cronJobs = append(cronJobs, weekend)
	return s, cronJobs
}

func echoTime() {
	log.Info("[echoTime]  echo time:%s", time.Now().Format("2006-01-02 15:04:05"))
}

func newPersonRequestParameter() interface{} {
	return &personRequestParam{}
}

func (c personRequestParam) String() string {
	b, _ := json.Marshal(c)
	return string(b)
}

func personHandle(_ *gin.Context, param interface{}) (interface{}, string) {
	p := param.(*personRequestParam)
	success := gin.H{
		"Code":           0,
		"Received Name":  p.Name,
		"Key ID":         p.KeyID,
		"Request Method": p.Method,
	}
	return success, p.String()
}

func restfulPostVpc(_ *gin.Context, param interface{}) (interface{}, string) {
	p := param.(*personRequestParam)
	success := gin.H{
		"Code":           0,
		"Request Method": p.Method,
		"Handle":         "POST /v2/vpc",
	}
	return success, p.String()
}
func restfulDeleteVpc(_ *gin.Context, param interface{}) (interface{}, string) {
	p := param.(*personRequestParam)
	success := gin.H{
		"Code":           0,
		"Request Method": p.Method,
		"Handle":         "DELETE /v2/vpc/:id",
	}
	return success, p.String()
}

func restfulPostSubnet(_ *gin.Context, param interface{}) (interface{}, string) {
	p := param.(*personRequestParam)
	success := gin.H{
		"Code":           0,
		"Request Method": p.Method,
		"Handle":         "POST /v2/vpc/subnet",
	}
	return success, p.String()
}

func restfulDeleteSubnet(_ *gin.Context, param interface{}) (interface{}, string) {
	p := param.(*personRequestParam)
	success := gin.H{
		"Code":           0,
		"Request Method": p.Method,
		"Handle":         "DELETE /v2/vpc/subnet/:id",
	}
	return success, p.String()
}

func registerHTTPHandles() {
	api.SetDefaultRestfulBindError(true, BadRequest)
	api.AddHTTPHandle2("/login", "Login", newLoginRequestParam, loginHandle)
	api.AddHTTPHandle2("/rand", "Rand", newRandRequestParam, randHandle)
	api.AddRESTFulAPIHttpHandle2("/person/:id", newPersonRequestParameter, personHandle, "HttpCode")
	api.AddRESTFulAPIHttpHandle("/three/:id/pay", newPersonRequestParameter, personHandle)
	api.AddHTTPHandle2("/LongTimerTask", "LongTimerTask", newLongTimerTaskParam, doLongTimerTask)
	api.AddHTTPHandle2("/LearnCode", "LearnCode", newLearnCodeParam, doLearnCodeTask)
	api.AddHTTPHandle2("/CallServiceByServiceName", "CallServiceByServiceName", newTestCheckKeepaliveRequest, doTestCheckKeepalive)
	api.AddHTTPHandle2("/CheckServiceCall", "CheckServiceCall", newCallServiceNameRequest, doCallServiceName)

	api.AddRESTFulAPIHttpHandle3("/v2/vpc", newPersonRequestParameter, restfulPostVpc, "HttpCode", "POST")
	api.AddRESTFulAPIHttpHandle3("/v2/vpc/:id", newPersonRequestParameter, restfulDeleteVpc, "HttpCode", "DELETE")
	api.AddRESTFulAPIHttpHandle3("/v2/vpc/subnet", newPersonRequestParameter, restfulPostSubnet, "HttpCode", "POST")
	api.AddRESTFulAPIHttpHandle3("/v2/vpc/subnet/:id", newPersonRequestParameter, restfulDeleteSubnet, "HttpCode", "DELETE")
}

func registerHTTPCustomHandles(router *gin.Engine) {
	f := func(c *gin.Context) {
		log.Info("my custom handle")
	}
	router.GET("/my", f)
}

func registerRGPCHandle(_ *grpc.Server) {

}
