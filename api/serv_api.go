package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/NeilXu2017/landau/data"
	"io"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/NeilXu2017/landau/log"
	"github.com/NeilXu2017/landau/prometheus"
	"github.com/NeilXu2017/landau/util"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/render"
)

type (
	//HTTPACLResult 权限检查结果
	HTTPACLResult int
	//HTTPHandleFunc URL处理程序
	HTTPHandleFunc func(c *gin.Context, param interface{}) (interface{}, string)
	//HTTPRequestParameter 构造http请求参数结构体
	HTTPRequestParameter func() interface{}
	//HTTPCheckACL 权限访问控制，返回非0值，则拒绝访问
	HTTPCheckACL func(urlPath string, actionID string, c *gin.Context) HTTPACLResult
	//HTTPCustomLogTag 自定义生成日志Tag
	HTTPCustomLogTag func(c *gin.Context) string
	//HTTPLogResponse HTTP处理结果日志内容
	HTTPLogResponse        func(response interface{}) string
	httpRequestActionParam struct {
		Action string `form:"Action" json:"Action" binding:"required"`
	}
	httpHandleEntry struct {
		handleFunc            HTTPHandleFunc
		newRequesterParameter HTTPRequestParameter
		logResponse           HTTPLogResponse
		logger                string
		httpCodeStatus        string
	}
	// HTTPAuditLog 审核日志记录
	HTTPAuditLog             func(urlPath string, action string, request *string, response *string, c *gin.Context)
	UnHtmlEscapeJsonResponse struct {
		Response interface{}
	}
)

const (
	//HTTPAclOK 有权限访问
	HTTPAclOK HTTPACLResult = iota
	//HTTPAclDeny 未识别到用户身份
	HTTPAclDeny
	//HTTPAclNoRight 用户无权限访问
	HTTPAclNoRight
)

var (
	httpEntry                      map[string]httpHandleEntry
	httpActionEntry                map[string]httpHandleEntry
	httpNeedCheckACL               = false
	httpCheckACL                   HTTPCheckACL
	httpAccessControlAllowOrigin   = true
	responseJSONPEnable            = true
	responseJSONPCallbackQueryName = "callback"
	customAPILogTag                = false
	httpCustomLogTag               HTTPCustomLogTag
	defaultAPILogger               = "API"
	defaultResponseLogAsJSON       = true
	defaultResponseShowDetail      = false
	defaultResponseShowSize        = 512
	defaultLogResponse             = func(response interface{}) string {
		msg := fmt.Sprintf("%v", response)
		if defaultResponseShowDetail {
			return msg
		}
		if utf8.RuneCountInString(msg) <= defaultResponseShowSize {
			return msg
		}
		responseRune := []rune(msg)
		return fmt.Sprintf("%s......%s", string(responseRune[0:64]), string(responseRune[len(responseRune)-64:]))
	}
	defaultAccessDenyResponse    = gin.H{"RetCode": 100, "Message": "请先登录"}
	defaultAccessNoRightResponse = gin.H{"RetCode": 101, "Message": "没有权限，请向管理员申请权限"}
	defaultBindErrorResponse     = gin.H{"RetCode": 160, "Message": "Missing Action"}
	httpAuditLog                 HTTPAuditLog
	postBindingComplexURL        map[string]string
	postBindingComplexAction     map[string]string
	unHtmlEscapeURL              = make(map[string]string)
	unHtmlEscapeAction           = make(map[string]string)
	unRegisterHandle             HTTPHandleFunc
	DisableTraceServiceAddress   bool
	replaceDefaultBindError      bool
	defaultBindErrorResponse2    interface{}
)

func (c *httpRequestActionParam) String() string {
	return fmt.Sprintf(`{"Action":"%s"}`, c.Action)
}

func (c UnHtmlEscapeJsonResponse) Render(w http.ResponseWriter) error {
	jsonEncoder := json.NewEncoder(w)
	jsonEncoder.SetEscapeHTML(false)
	return jsonEncoder.Encode(c.Response)
}

func (c UnHtmlEscapeJsonResponse) WriteContentType(w http.ResponseWriter) {
	header := w.Header()
	header["Content-Type"] = []string{"application/json; charset=utf-8"}
}

func JsonMarshalNoEscapeHTML(d interface{}) ([]byte, error) {
	b := &bytes.Buffer{}
	jsonEncoder := json.NewEncoder(b)
	jsonEncoder.SetEscapeHTML(false)
	err := jsonEncoder.Encode(d)
	return b.Bytes(), err
}

func init() {
	httpEntry = make(map[string]httpHandleEntry)
	httpActionEntry = make(map[string]httpHandleEntry)
	newActionParam := func() interface{} {
		return &httpRequestActionParam{}
	}
	AddHTTPHandle2("/", "", newActionParam, dispatchAction)
}

func SetDefaultBindError(replaced bool, replaceResponse interface{}) {
	replaceDefaultBindError = replaced
	defaultBindErrorResponse2 = replaceResponse
}

// SetHTTPAuditLog 设置审计日志记录回调函数
func SetHTTPAuditLog(auditLog HTTPAuditLog) {
	httpAuditLog = auditLog
}

// SetDefaultHTTPLogResponse 设置缺省logResponse参数
func SetDefaultHTTPLogResponse(showDetail bool, showSize int) {
	defaultResponseShowDetail = showDetail
	defaultResponseShowSize = showSize
}

// SetHTTPCheckACL 设置权限检查参数
func SetHTTPCheckACL(needCheckACL bool, checkACL HTTPCheckACL) {
	httpNeedCheckACL = needCheckACL
	httpCheckACL = checkACL
}

// SetHTTPAccessControlAllowOrigin 是否设置httpAccessControlAllowOrigin头
func SetHTTPAccessControlAllowOrigin(accessControlAllowOrigin bool) {
	httpAccessControlAllowOrigin = accessControlAllowOrigin
}

// SetResponseJSONP 设置JSONP参数
func SetResponseJSONP(jsonpEnable bool, jsonpCallbackQueryName string) {
	responseJSONPEnable = jsonpEnable
	responseJSONPCallbackQueryName = jsonpCallbackQueryName
}

// SetHTTPCustomLogTag 设置自定义Log tag 参数
func SetHTTPCustomLogTag(enableCustomLogTag bool, customLogTag HTTPCustomLogTag) {
	customAPILogTag = enableCustomLogTag
	httpCustomLogTag = customLogTag
}

// SetResponseLogAsJSON 设置日志记录Response是否以json格式化，缺省是
func SetResponseLogAsJSON(responseLogAsJSON bool) {
	defaultResponseLogAsJSON = responseLogAsJSON
}

// SetUnRegisterHandle 设置未注册的Action处理入口
func SetUnRegisterHandle(handle HTTPHandleFunc) {
	unRegisterHandle = handle
}

// AddHTTPHandle 注册URL处理程序
func AddHTTPHandle(urlPath string, actionID string, newRequesterParameter HTTPRequestParameter, handleFunc HTTPHandleFunc, logResponse HTTPLogResponse, loggerName string) {
	h := httpHandleEntry{
		handleFunc:            handleFunc,
		newRequesterParameter: newRequesterParameter,
		logResponse:           logResponse,
		logger:                loggerName,
	}
	if urlPath != "" {
		httpEntry[urlPath] = h
	}
	if actionID != "" {
		httpActionEntry[actionID] = h
	}
}

// AddHTTPHandle2 注册URL处理程序
func AddHTTPHandle2(urlPath string, actionID string, newRequesterParameter HTTPRequestParameter, handleFunc HTTPHandleFunc) {
	AddHTTPHandle(urlPath, actionID, newRequesterParameter, handleFunc, defaultLogResponse, defaultAPILogger)
}

func AddHTTPHandle3(urlPath string, actionID string, newRequesterParameter HTTPRequestParameter, handleFunc HTTPHandleFunc, httpCodeStatus string) {
	h := httpHandleEntry{
		handleFunc:            handleFunc,
		newRequesterParameter: newRequesterParameter,
		logResponse:           defaultLogResponse,
		logger:                defaultAPILogger,
		httpCodeStatus:        httpCodeStatus,
	}
	if urlPath != "" {
		httpEntry[urlPath] = h
	}
	if actionID != "" {
		httpActionEntry[actionID] = h
	}
}

// AddUnHtmlEscapeHttpHandle 注册URL处理程序,响应json内容不进行 HTML Escape 处理
func AddUnHtmlEscapeHttpHandle(urlPath string, actionID string, newRequesterParameter HTTPRequestParameter, handleFunc HTTPHandleFunc) {
	AddHTTPHandle(urlPath, actionID, newRequesterParameter, handleFunc, defaultLogResponse, defaultAPILogger)
	if urlPath != "" && urlPath != "/" {
		unHtmlEscapeURL[urlPath] = "1"
	}
	if actionID != "" {
		unHtmlEscapeAction[actionID] = "1"
	}
}

// RegisterHTTPHandle 向gin.Engine注册URL处理入口
func RegisterHTTPHandle(r *gin.Engine) {
	for k := range httpEntry {
		r.GET(k, httpHandleProxy)
		r.POST(k, httpHandleProxy)
	}
	optionResponse := func(c *gin.Context) {
		if v, ok := c.Request.Header["Origin"]; ok && len(v) > 0 && v[0] != "" {
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
			c.Header("Access-Control-Allow-Origin", v[0])
			c.Header("Access-Control-Allow-Headers", "Content-Type")
		}
		success := gin.H{
			"RetCode": 0,
			"Message": "options success",
		}
		c.JSON(http.StatusOK, success)
	}
	r.NoRoute(func(c *gin.Context) {
		if c.Request.Method == "OPTIONS" {
			optionResponse(c)
			return
		}
		start := time.Now()
		urlPath := c.Request.URL.Path
		addAccessControlAllowHeader(c)
		if unRegisterHandle == nil {
			c.String(http.StatusNotFound, "404 page not found:%s", urlPath)
			return
		}
		p := &httpRequestActionParam{}
		isPostMethod := c.Request.Method == "POST"
		isBindingComplex := isPostBindingComplex(urlPath, "")
		_, _ = bindParams(c, &p, isPostMethod, isBindingComplex)
		if response, isDeny := isACLDeny(urlPath, p.Action, c); isDeny {
			c.JSON(http.StatusOK, response)
			return
		}
		response, requestParamLog := unRegisterHandle(c, p)
		c.JSON(http.StatusOK, response)
		strCustomLogTag := ""
		if customAPILogTag && httpCustomLogTag != nil {
			strCustomLogTag = httpCustomLogTag(c)
		}
		strResponse := ""
		if defaultResponseLogAsJSON {
			byteResp, _ := json.Marshal(response)
			strResponse = string(byteResp)
		} else {
			strResponse = fmt.Sprintf("%v", response)
		}
		if httpAuditLog != nil {
			actionName, bizResponse := getHTTPAuditLogContent(urlPath, p, response)
			httpAuditLog(urlPath, actionName, &requestParamLog, &bizResponse, c)
		}
		log.Info2("API", "[%s]\t[%s]\t%s\tRequest:%s\tResponse:%v", urlPath, time.Since(start), strCustomLogTag, requestParamLog, defaultLogResponse(strResponse))
	})
}

func addAccessControlAllowHeader(c *gin.Context) {
	if httpAccessControlAllowOrigin {
		if v, ok := c.Request.Header["Origin"]; ok && len(v) > 0 && v[0] != "" {
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
			c.Header("Access-Control-Allow-Origin", v[0])
		}
	}
}

func isACLDeny(urlPath string, actionID string, c *gin.Context) (interface{}, bool) {
	if httpNeedCheckACL && httpCheckACL != nil {
		aclResult := httpCheckACL(urlPath, actionID, c)
		switch aclResult {
		case HTTPAclDeny:
			return defaultAccessDenyResponse, true
		case HTTPAclNoRight:
			return defaultAccessNoRightResponse, true
		}
	}
	return nil, false
}

func dispatchAction(c *gin.Context, requestParams interface{}) (interface{}, string) {
	p := requestParams.(*httpRequestActionParam)
	_traceLastServiceAddress(c)
	if a, existed := httpActionEntry[p.Action]; existed {
		if response, isDeny := isACLDeny("/", p.Action, c); isDeny {
			return response, ""
		}
		isPostMethod := c.Request.Method == "POST"
		bizParamStruct := a.newRequesterParameter()
		isBindingComplex := isPostBindingComplex("", p.Action)
		param, bindError := bindParams(c, &bizParamStruct, isPostMethod, isBindingComplex)
		if bindError == nil {
			rsp, reqStr := a.handleFunc(c, param)
			return rsp, reqStr
		}
		return gin.H{"RetCode": 230, "Message": fmt.Sprintf("Bind params error [%v]", bindError)}, p.String()
	}
	if unRegisterHandle != nil {
		if response, isDeny := isACLDeny("/", p.Action, c); isDeny {
			return response, ""
		}
		rsp, reqStr := unRegisterHandle(c, requestParams)
		return rsp, reqStr
	}
	return defaultBindErrorResponse, p.String()
}

func bindParams(c *gin.Context, param interface{}, isPostMethod bool, isBindingComplex bool) (interface{}, error) {
	var bindError error
	v := reflect.ValueOf(param)
	p := reflect.Indirect(v).Interface()
	if isPostMethod {
		bindError = bindPost(p, c, isBindingComplex)
	} else {
		bindError = bindQuery(p, c)
	}
	return p, bindError
}

func bindParamsRestful(c *gin.Context, param interface{}, isPostMethod bool, isBindingComplex bool, id, method string) (interface{}, error) {
	var bindError error
	v := reflect.ValueOf(param)
	p := reflect.Indirect(v).Interface()
	setRestFulKeys(p, id, method)
	if isPostMethod {
		bindError = bindPost(p, c, isBindingComplex)
	} else {
		bindError = bindQuery(p, c)
	}
	return p, bindError
}
func httpHandleProxy(c *gin.Context) {
	start := time.Now()
	urlPath := c.Request.URL.Path
	isPostMethod := c.Request.Method == "POST"
	_traceLastServiceAddress(c)
	if isPostMethod { //POST 将 Request.Body 对象转换成可重复读取对象
		var bodyBytes []byte
		if c.Request.Body != nil {
			bodyBytes, _ = io.ReadAll(c.Request.Body)
		}
		c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	}
	isBindingComplex := isPostBindingComplex(urlPath, "")
	if a, existed := httpEntry[urlPath]; existed {
		if urlPath != "/" {
			if response, isDeny := isACLDeny(urlPath, "", c); isDeny {
				jsonpCallback := ""
				if responseJSONPEnable {
					jsonpCallback = c.DefaultQuery(responseJSONPCallbackQueryName, "")
				}
				addAccessControlAllowHeader(c)
				httpCode := http.StatusOK
				if a.httpCodeStatus != "" {
					httpCode = getHttpStatusCodeFromResponseObject(response, a.httpCodeStatus, http.StatusOK)
				}
				if jsonpCallback == "" {
					c.JSON(httpCode, response)
				} else {
					c.Render(httpCode, render.JsonpJSON{Callback: jsonpCallback, Data: response})
				}
				strCustomLogTag := ""
				if customAPILogTag && httpCustomLogTag != nil {
					strCustomLogTag = httpCustomLogTag(c)
				}
				strResponse := ""
				if defaultResponseLogAsJSON {
					byteResp, _ := json.Marshal(response)
					strResponse = string(byteResp)
				} else {
					strResponse = fmt.Sprintf("%v", response)
				}
				log.Info2(a.logger, "[%s]\t[%s]\t%s\tRequest:%s\tResponse:%v", urlPath, time.Since(start), strCustomLogTag, "{}", strResponse)
				prometheus.UpdateApiMetric(getRetCodeFromInterface(response), "", start, c.Request, "")
				return
			}
		}
		prepareRequestParam(c, isPostMethod)
		var response interface{}
		requestParamLog := ""
		bizParamStruct := a.newRequesterParameter()
		param, bindError := bindParams(c, &bizParamStruct, isPostMethod, isBindingComplex)
		if bindError == nil {
			response, requestParamLog = a.handleFunc(c, param)
		} else {
			if replaceDefaultBindError {
				response = defaultBindErrorResponse2
			} else {
				response = gin.H{"RetCode": 230, "Message": fmt.Sprintf("Bind params error [%v]", bindError)}
			}
		}
		jsonpCallback := ""
		if responseJSONPEnable {
			jsonpCallback = c.DefaultQuery(responseJSONPCallbackQueryName, "")
		}
		addAccessControlAllowHeader(c)
		jsonEscapeHtml := true
		if urlPath == "/" {
			if p, ok := param.(*httpRequestActionParam); ok {
				if _, ok := unHtmlEscapeAction[p.Action]; ok {
					jsonEscapeHtml = false
				}
			}
		} else {
			if _, ok := unHtmlEscapeURL[urlPath]; ok {
				jsonEscapeHtml = false
			}
		}
		httpCode := http.StatusOK
		if a.httpCodeStatus != "" {
			httpCode = getHttpStatusCodeFromResponseObject(response, a.httpCodeStatus, http.StatusOK)
		}
		if jsonpCallback == "" {
			if jsonEscapeHtml {
				c.JSON(httpCode, response)
			} else {
				noEscapeHtmlResponse := UnHtmlEscapeJsonResponse{Response: response}
				c.Render(httpCode, noEscapeHtmlResponse)
			}
		} else {
			c.Render(httpCode, render.JsonpJSON{Callback: jsonpCallback, Data: response})
		}
		strCustomLogTag := ""
		if customAPILogTag && httpCustomLogTag != nil {
			strCustomLogTag = httpCustomLogTag(c)
		}
		strResponse := ""
		if defaultResponseLogAsJSON {
			if jsonEscapeHtml {
				byteResp, _ := json.Marshal(response)
				strResponse = string(byteResp)
			} else {
				byteResp, _ := JsonMarshalNoEscapeHTML(response)
				strResponse = string(byteResp)
			}
		} else {
			strResponse = fmt.Sprintf("%v", response)
		}
		if httpAuditLog != nil {
			actionName, bizResponse := getHTTPAuditLogContent(urlPath, param, response)
			httpAuditLog(urlPath, actionName, &requestParamLog, &bizResponse, c)
		}
		urlLogResponse, apiLogger := a.logResponse, a.logger
		if urlPath == "/" {
			if p, ok := param.(*httpRequestActionParam); ok {
				if actionEntry, actionOK := httpActionEntry[p.Action]; actionOK {
					urlLogResponse = actionEntry.logResponse
					if actionEntry.logger != "" {
						apiLogger = actionEntry.logger
					}
				}
			}
		}
		log.Info2(apiLogger, "[%s]\t[%s]\t%s\tRequest:%s\tResponse:%v", urlPath, time.Since(start), strCustomLogTag, requestParamLog, urlLogResponse(strResponse))
		prometheus.UpdateApiMetric(getRetCodeFromInterface(response), getActionFromInterface(param), start, c.Request, "")
	}
}

func getHTTPAuditLogContent(url string, request interface{}, response interface{}) (string, string) {
	action, rsp := "", ""
	if url == "/" {
		actionParam := httpRequestActionParam{}
		_ = util.Copy(&actionParam, request)
		action = actionParam.Action
	}
	if b, err := json.Marshal(response); err == nil {
		rsp = string(b)
	}
	return action, rsp
}

// SetPostBindingComplex 设置需要设置复杂JSON Unmarshal的Action
func SetPostBindingComplex(val [2]string) {
	postBindingComplexURL = make(map[string]string)
	postBindingComplexAction = make(map[string]string)
	for _, s := range strings.Split(val[0], ",") {
		if url := strings.TrimSpace(s); url != "" {
			postBindingComplexURL[url] = ""
		}
	}
	for _, s := range strings.Split(val[1], ",") {
		if action := strings.TrimSpace(s); action != "" {
			postBindingComplexAction[action] = ""
		}
	}
}

func getActionFromInterface(v interface{}) string {
	if p, ok := v.(*httpRequestActionParam); ok {
		return p.Action
	}
	val := reflect.ValueOf(v).Elem()
	if val.Kind() == reflect.Struct {
		typ := reflect.TypeOf(v).Elem()
		for i := 0; i < typ.NumField(); i++ {
			if typeField := typ.Field(i); typeField.Name == "Action" {
				actionVal := val.Field(i)
				switch actionVal.Kind() {
				case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
					return strconv.Itoa(int(actionVal.Int()))
				case reflect.Float32, reflect.Float64:
					return fmt.Sprintf("%v", actionVal.Float())
				default:
					if actionVal.CanInterface() {
						v := actionVal.Interface()
						if vStr, ok := v.(fmt.Stringer); ok {
							return vStr.String()
						} else {
							return fmt.Sprintf("%v", v)
						}
					}
				}
			}
		}
	}
	return ""
}

func getRetCodeFromInterface(v interface{}) int {
	if gH, ok := v.(gin.H); ok {
		if v, ok := gH["RetCode"]; ok {
			return _getIntValueFromInterface(v)
		}
	}
	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Struct {
		typ := reflect.TypeOf(v)
		for i := 0; i < typ.NumField(); i++ {
			if typeField := typ.Field(i); typeField.Name == "RetCode" {
				retCodeVal := val.Field(i)
				switch retCodeVal.Kind() {
				case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
					return int(retCodeVal.Int())
				case reflect.Float32, reflect.Float64:
					return int(retCodeVal.Float())
				default:
					if retCodeVal.CanInterface() {
						return _getIntValueFromInterface(retCodeVal.Interface())
					}
				}
			}
		}
	}
	return 0
}

func _getIntValueFromInterface(v interface{}) int {
	switch vt := v.(type) {
	case int:
		return vt
	case float64:
		return int(vt)
	case string:
		if iv, err := strconv.Atoi(vt); err == nil {
			return iv
		}
	}
	return 0
}

func _traceLastServiceAddress(c *gin.Context) {
	if !DisableTraceServiceAddress {
		serviceName := c.Request.Header.Get(data.ServiceNameHeadTag)
		address := c.Request.Header.Get(data.ServiceAddressHeadTag)
		if serviceName != "" && address != "" {
			data.LastTraceServiceAddress.Store(serviceName, address)
		}
	}
}
