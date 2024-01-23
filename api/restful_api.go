package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/NeilXu2017/landau/log"
	"github.com/NeilXu2017/landau/prometheus"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/render"
)

type (
	_RESTFulApiEntry struct {
		Url                 string
		NewRequestParameter HTTPRequestParameter
		HttpHandle          HTTPHandleFunc
		_urls               []string
		ID                  string
		HttpCodeStatus      string
		UrlRegex            *regexp.Regexp
		HttpMethod          string
	}
)

var (
	restFulHttpEntry                = make(map[string]_RESTFulApiEntry)
	replaceDefaultRestfulBindError  bool
	defaultRestfulBindErrorResponse interface{}
)

func AddRESTFulAPIHttpHandle(urlPath string, newRequesterParameter HTTPRequestParameter, handleFunc HTTPHandleFunc) {
	AddRESTFulAPIHttpHandle3(urlPath, newRequesterParameter, handleFunc, "", "")
}

func AddRESTFulAPIHttpHandle2(urlPath string, newRequesterParameter HTTPRequestParameter, handleFunc HTTPHandleFunc, httpCodeFieldName string) {
	AddRESTFulAPIHttpHandle3(urlPath, newRequesterParameter, handleFunc, httpCodeFieldName, "")
}
func AddRESTFulAPIHttpHandle3(urlPath string, newRequesterParameter HTTPRequestParameter, handleFunc HTTPHandleFunc, httpCodeFieldName string, httpMethod string) {
	urls, id := strings.Split(urlPath, "/"), ""
	var keyUrl []string
	for _, u := range urls {
		if strings.Index(u, ":") == 0 {
			id = strings.Replace(u, ":", "", 1)
			keyUrl = append(keyUrl, "[^/]*")
		} else {
			keyUrl = append(keyUrl, u)
		}
	}
	restFulHttpEntry[urlPath] = _RESTFulApiEntry{
		Url:                 urlPath,
		NewRequestParameter: newRequesterParameter,
		HttpHandle:          handleFunc,
		_urls:               urls,
		ID:                  id,
		HttpCodeStatus:      httpCodeFieldName,
		UrlRegex:            regexp.MustCompile(fmt.Sprintf("^%s$", strings.Join(keyUrl, "/"))),
		HttpMethod:          strings.ToUpper(httpMethod),
	}
}

func SetDefaultRestfulBindError(replaced bool, replaceResponse interface{}) {
	replaceDefaultRestfulBindError = replaced
	defaultRestfulBindErrorResponse = replaceResponse
}

func isExistRESTFul(urlPath string, httpMethod string) (*_RESTFulApiEntry, bool) {
	requestURL := strings.Split(urlPath, "/")
	var matchedEntry []_RESTFulApiEntry
	for _, a := range restFulHttpEntry {
		if (a.HttpMethod == "" || a.HttpMethod == httpMethod) && a.UrlRegex.MatchString(urlPath) && len(requestURL) == len(a._urls) {
			isMatched := true
			for i, j := 0, len(requestURL); i < j; i++ {
				if requestURL[i] != a._urls[i] && strings.Index(a._urls[i], ":") != 0 {
					isMatched = false
					break
				}
			}
			if isMatched {
				if a.ID == "" { //no id has more high priority
					return &a, true
				}
				matchedEntry = append(matchedEntry, a)
			}
		}
	}
	if len(matchedEntry) > 0 {
		if len(matchedEntry) == 1 {
			return &matchedEntry[0], true
		}
		log.Error2("[isExistRESTFul] requestUrl:%s have more than one matched,please check api entry:%v", urlPath, matchedEntry)
	}
	return nil, false
}

func setRestFulKeys(ptr interface{}, ID, method string) {
	typ := reflect.TypeOf(ptr).Elem()
	val := reflect.ValueOf(ptr).Elem()
	for i := 0; i < typ.NumField(); i++ {
		typeField := typ.Field(i)
		structField := val.Field(i)
		if !structField.CanSet() {
			continue
		}
		inputFieldName := typeField.Tag.Get("restful")
		switch inputFieldName {
		case "id":
			_ = setWithProperType(typeField.Type.Kind(), ID, structField)
		case "method":
			_ = setWithProperType(typeField.Type.Kind(), method, structField)
		}
	}
}

func restFullHttpHandleProxy(c *gin.Context) {
	urlPath := c.Request.URL.Path
	var bodyBytes []byte
	if c.Request.Body != nil {
		bodyBytes, _ = io.ReadAll(c.Request.Body)
		c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	}
	isBindingComplex := isPostBindingComplex(urlPath, "")
	isPostMethod := c.Request.Method != "GET"
	if a, existed := isExistRESTFul(urlPath, c.Request.Method); existed {
		start := time.Now()
		prepareRequestParam(c, isPostMethod)
		var response interface{}
		requestParamLog := ""
		bizParamStruct := a.NewRequestParameter()
		param, bindError := bindParamsRestful(c, &bizParamStruct, isPostMethod, isBindingComplex, c.Param(a.ID), c.Request.Method)
		if bindError == nil {
			response, requestParamLog = a.HttpHandle(c, param)
		} else {
			if replaceDefaultRestfulBindError {
				response = defaultRestfulBindErrorResponse
			} else {
				response = gin.H{"RetCode": 230, "Message": fmt.Sprintf("Bind params error [%v]", bindError)}
			}
		}
		jsonpCallback := ""
		if responseJSONPEnable {
			jsonpCallback = c.DefaultQuery(responseJSONPCallbackQueryName, "")
		}
		addAccessControlAllowHeader(c)
		httpCode := http.StatusOK
		if a.HttpCodeStatus != "" {
			httpCode = getHttpStatusCodeFromResponseObject(response, a.HttpCodeStatus, http.StatusOK)
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
		log.Info2(defaultAPILogger, "[%s]\t[%s]\t%s\tRequest:%s\tResponse:%v", urlPath, time.Since(start), strCustomLogTag, requestParamLog, defaultLogResponse(strResponse))
		prometheus.UpdateApiMetric(getRetCodeFromInterface(response), getActionFromInterface(param), start, c.Request, a.Url)
	}
}

// RegisterRestfulHTTPHandle 向gin.Engine注册URL处理入口
func RegisterRestfulHTTPHandle(r *gin.Engine) {
	for k := range restFulHttpEntry {
		r.Any(k, restFullHttpHandleProxy)
	}
}

func getHttpStatusCodeFromResponseObject(ptr interface{}, fieldName string, defaultHttpCode int) int {
	if gH, ok := ptr.(gin.H); ok {
		if v, ok := gH[fieldName]; ok {
			return _getIntValueFromInterface(v)
		}
	}
	val := reflect.ValueOf(ptr)
	switch val.Kind() {
	case reflect.Struct:
		typ := reflect.TypeOf(ptr)
		for i := 0; i < typ.NumField(); i++ {
			if typeField := typ.Field(i); typeField.Name == fieldName {
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
	case reflect.Map:
		if m, ok := ptr.(map[string]interface{}); ok {
			if v, ok := m[fieldName]; ok {
				return _getIntValueFromInterface(v)
			}
		}
	}
	return defaultHttpCode
}
