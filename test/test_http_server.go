package test

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/NeilXu2017/landau/data"

	"github.com/NeilXu2017/landau/web"

	"github.com/NeilXu2017/landau/util"

	"github.com/NeilXu2017/landau/api"
	"github.com/NeilXu2017/landau/log"
	"github.com/gin-gonic/gin"
)

type (
	_ResourceParameter struct {
		ResID   string `form:"res_id" json:"res_id"`
		ResName string `form:"res_name" json:"res_name"`
	}
	_MetricFieldParameter struct {
		FieldName string `form:"name" json:"name"`
		FieldUnit string `form:"unit" json:"unit"`
		DataType  string `form:"data_type" json:"data_type"`
		InitMin   int    `form:"init_min" json:"init_min"`
		InitMax   int    `form:"init_max" json:"init_max"`
		TagName   string `form:"tag_name" json:"tag_name"`
		TagValue  string `form:"tag_value" json:"tag_value"`
	}

	loginRequestParam struct {
		UserID    string                  `form:"user_id" json:"user_id" binding:"required"`
		Password  string                  `form:"user_pwd" json:"user_pwd" binding:"required"`
		Action    string                  `json:"Action"`
		Resources []_ResourceParameter    `form:"resources" json:"resources"`
		Metrics   []_MetricFieldParameter `form:"metrics" json:"metrics"`
	}

	loginResponseParam struct {
		UserID    string `json:"user_id"`
		UserName  string `json:"user_name"`
		LoginTime string `json:"login_time"`
		Debug     string `json:"debug"`
	}
	randRequestParam struct {
		ID     int    `form:"num" json:"num"`
		Action string `json:"Action"`
	}
	personRequestParam struct {
		KeyID  int    `restful:"id"`
		Method string `restful:"method"`
		Name   string `form:"name" json:"name"`
	}
	ResponseAsStruct struct {
		RetCode int
	}
	longTimerTaskParam struct {
		SleepTimes int `form:"sleep_times" json:"sleep_times"`
	}

	learnCodeParam struct {
		Action     string
		StackCount int
		Unit       int
	}
)

func (c *loginRequestParam) String() string {
	c.Action = "Login"
	b, _ := json.Marshal(c)
	return string(b)
}

func (c *randRequestParam) String() string {
	c.Action = "Rand"
	b, _ := json.Marshal(c)
	return string(b)
}

func (c *learnCodeParam) String() string {
	c.Action = "LearnCode"
	b, _ := json.Marshal(c)
	return string(b)
}

func newLoginRequestParam() interface{} {
	return &loginRequestParam{}
}

func newRandRequestParam() interface{} {
	return &randRequestParam{}
}

func newLearnCodeParam() interface{} {
	return &learnCodeParam{StackCount: 10, Unit: 10}
}

func loginHandle(c *gin.Context, param interface{}) (interface{}, string) {
	request := param.(*loginRequestParam)
	userName := ""
	retCode := 0
	retMessage := "login success"
	switch request.UserID {
	case "neil", "landau", "Neil", "Landau", "xuLong", "XuLong":
		userName = "张三"
	case "hello", "Hello":
		userName = "World"
	default:
		retCode = 100101
		retMessage = "login failure"
	}
	if retCode == 0 {
		sToken, _ := web.ReadCookie(c, "landau_session_id")
		if sToken == "" {
			sToken = fmt.Sprintf("%d_%d_%d", rand.Intn(100), rand.Intn(10), rand.Intn(1000))
			sToken = base64.StdEncoding.EncodeToString([]byte(sToken))
		}
		web.WriteCookie(c, "landau_session_id", sToken)
	}
	debugMsg := ""
	if len(request.Resources) > 0 {
		debugMsg = fmt.Sprintf("Resource ID:%s name:%s", request.Resources[0].ResID, request.Resources[0].ResName)
	}
	if len(request.Metrics) > 0 {
		debugMsg = fmt.Sprintf("%s Metric:%v", debugMsg, request.Metrics[0])
	}
	if request.UserID == "hello" || request.UserID == "Hello" {
		retCode := 0
		if request.UserID == "Hello" {
			retCode = 144
		}
		success := ResponseAsStruct{RetCode: retCode}
		return success, request.String()
	} else {
		success := gin.H{
			"RetCode":    retCode,
			"Message":    retMessage,
			"login_user": loginResponseParam{UserID: request.UserID, UserName: userName, LoginTime: time.Now().Format("2006-01-02 15:04:05")},
			"debug_msg":  debugMsg,
		}
		return success, request.String()
	}
}

func randHandle(_ *gin.Context, param interface{}) (interface{}, string) {
	request := param.(*randRequestParam)
	m := make(map[string]interface{})
	data.CallHTTPService2("http://10.64.205.36:9690/get_region_service_info", request, m)
	success := gin.H{
		"RetCode": 0,
		"Message": `rand success can optionally supply the name of a 可以有选kě yǐ择地提供自 custom dictionary and a Boolean value indicating whether you want to ignore case.
		您可以有选择地提供自定义字典的名称和一个指示是否忽略大小写的Boolean值。
		nín kě yǐ yǒu ǎn zé de tí gōng zì dìng yì zì diǎn de míng chēng hé yī gè zhǐ shì shì fǒu hū lüè dà ǎo xiě de Boolean zhí 。
		按钮使用按钮可让网站访问者在填完表单后提交表单、通过重置表单来清除各个域或者运行自定义脚本。
		specify a schedule for updating that page, and how much content to download, click Customize.
		标准库 time.Time 就实现了这两个接口。另外一个简单的例子（这个例子来自于参考资料中 Go and JSON 文章）：
type Month struct {
    MonthNumber int
    YearNumber int
}
		`,
		"id": request.ID + rand.Intn(1000),
	}
	return success, request.String()
}

func myCheckACL(urlPath string, actionID string, _ *gin.Context) api.HTTPACLResult {
	log.Info("[CheckACL] urlPath:%s Action:%s", urlPath, actionID)
	return api.HTTPAclOK
}

func getUserSessionID(c *gin.Context) string {
	sToken, _ := web.ReadCookie(c, "landau_session_id")
	return sToken
}

// CheckHTTPServer 测试Web服务
func CheckHTTPServer() {
	gin.DefaultWriter = log.NewConsoleLogger("gin")
	gin.DefaultErrorWriter = log.NewConsoleLogger("gin")
	router := gin.Default()
	api.AddHTTPHandle2("/login", "Login", newLoginRequestParam, loginHandle)
	api.AddHTTPHandle2("/rand", "Rand", newRandRequestParam, randHandle)
	api.RegisterHTTPHandle(router)
	api.SetHTTPCheckACL(true, myCheckACL)
	api.SetHTTPCustomLogTag(true, getUserSessionID)
	serverAddress := fmt.Sprintf("%s:%d", util.IPConvert("127.0.0.1", util.IPV6Bracket), 8080)
	log.Info("HTTP Service Address:%s", serverAddress)
	_ = router.Run(serverAddress)
}

// CheckHTTPClient 客户端调用
func CheckHTTPClient() {
	h1()
	h2()
	h3()
}

func h1() {
	param := make(map[string]interface{})
	param["num"] = "95"
	type checkResponse struct {
		Message string `json:"Message"`
		RetCode int    `json:"RetCode"`
		ID      int    `json:"id"`
	}
	c := checkResponse{}
	httpHelper, _ := data.NewHTTPHelper(
		data.SetHTTPMethod(data.HTTPGet),
		data.SetHTTPUrl("http://127.0.0.1:8080/rand"),
		data.SetHTTPRequestParams(param),
	)
	_ = httpHelper.Call2(&c)
	log.Info("CheckHTTPClient result:%v", c)
}

func h2() {
	param := make(map[string]interface{})
	param["num"] = "103"
	param["Action"] = "Rand"
	type checkResponse struct {
		Message string `json:"Message"`
		RetCode int    `json:"RetCode"`
		ID      int    `json:"id"`
	}
	c := checkResponse{}
	httpHelper, _ := data.NewHTTPHelper(
		data.SetHTTPUrl("http://127.0.0.1:8080/rand"),
		data.SetHTTPRequestParams(param),
	)
	_ = httpHelper.Call2(&c)
	log.Info("CheckHTTPClient result:%v", c)
}

func h3() {
	jsonRequest := randRequestParam{
		ID:     992,
		Action: "Rand",
	}
	type checkResponse struct {
		Message string `json:"Message"`
		RetCode int    `json:"RetCode"`
		ID      int    `json:"id"`
	}
	customLogRequest := func(p interface{}) string {
		if v, ok := p.(randRequestParam); ok {
			return fmt.Sprintf("Custom log request, skip action, ID=%d", v.ID)
		}
		return fmt.Sprintf("%v", p)
	}
	c := checkResponse{}
	httpHelper, _ := data.NewHTTPHelper(
		data.SetHTTPUrl("http://127.0.0.1:8080/rand"),
		data.SetHTTPRequestRawObject(jsonRequest),
		data.SetHTTPRequestLogger(customLogRequest),
	)
	_ = httpHelper.Call2(&c)
	log.Info("CheckHTTPClient result:%+v", c)
}

func (c *longTimerTaskParam) String() string {
	b, _ := json.Marshal(c)
	return string(b)
}

func doLongTimerTask(_ *gin.Context, param interface{}) (interface{}, string) {
	t := time.Now()
	request := param.(*longTimerTaskParam)
	if request.SleepTimes > 0 {
		time.Sleep(time.Second * time.Duration(request.SleepTimes))
	}
	success := gin.H{"RetCode": 0, "Message": "LongTimerTaskResponse", "TimeDuration": time.Since(t)}
	return success, request.String()
}

func newLongTimerTaskParam() interface{} {
	return &longTimerTaskParam{}
}

func doLearnCodeTask(_ *gin.Context, param interface{}) (interface{}, string) {
	request := param.(*learnCodeParam)
	unit := request.Unit
	if unit <= 1 {
		unit = 1
	}
	sleepTime := time.Duration((1+rand.Intn(15))*unit) * time.Millisecond
	s1, s2 := checkCost(sleepTime, request.StackCount)
	success := gin.H{"RetCode": 0, "Message": "LearCodeResponse", "PanicCost": s1, "NoPanicCost": s2}
	return success, request.String()
}

func checkCost(s time.Duration, stackCount int) (string, string) {
	t := time.Now()
	var s1, s2 string
	wg := sync.WaitGroup{}
	wg.Add(2)
	go func() {
		callCost(s, stackCount, true)
		s1 = time.Since(t).String()
		wg.Done()
	}()
	go func() {
		callCost(s, stackCount, false)
		s2 = time.Since(t).String()
		wg.Done()
	}()
	wg.Wait()
	return s1, s2
}

var (
	ErrAbort = errors.New("abort error")
)

func callCost(s time.Duration, stackCount int, endPanic bool) {
	defer func() {
		if err := recover(); err != nil && err != ErrAbort {
			fmt.Printf("Panic:%v\n", err)
		}
	}()
	callFuncStackCount(s, stackCount, endPanic)
}

func callFuncStackCount(s time.Duration, stackCount int, endPanic bool) {
	if stackCount <= 0 {
		time.Sleep(s)
		if endPanic {
			panic(ErrAbort)
		}
		return
	}
	callFuncStackCount(s, stackCount-1, endPanic)
}
