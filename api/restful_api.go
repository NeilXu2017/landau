package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
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
	}
)

var (
	restFulHttpEntry = make(map[string]_RESTFulApiEntry)
)

func AddRESTFulAPIHttpHandle(urlPath string, newRequesterParameter HTTPRequestParameter, handleFunc HTTPHandleFunc) {
	urls, id := strings.Split(urlPath, "/"), ""
	for _, u := range urls {
		if strings.Index(u, ":") == 0 {
			id = strings.Replace(u, ":", "", 1)
			break
		}
	}
	restFulHttpEntry[urlPath] = _RESTFulApiEntry{Url: urlPath, NewRequestParameter: newRequesterParameter, HttpHandle: handleFunc, _urls: urls, ID: id}
}

func isExistRESTFul(urlPath string) (*_RESTFulApiEntry, bool) {
	requestURL := strings.Split(urlPath, "/")
	for _, a := range restFulHttpEntry {
		if len(requestURL) == len(a._urls) {
			isMatched := true
			for i, j := 0, len(requestURL); i < j; i++ {
				if requestURL[i] != a._urls[i] && strings.Index(a._urls[i], ":") != 0 {
					isMatched = false
					break
				}
			}
			if isMatched {
				return &a, true
			}
		}
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
	if a, existed := isExistRESTFul(urlPath); existed {
		start := time.Now()
		prepareRequestParam(c, isPostMethod)
		var response interface{}
		requestParamLog := ""
		bizParamStruct := a.NewRequestParameter()
		param, bindError := bindParamsRestful(c, &bizParamStruct, isPostMethod, isBindingComplex, c.Param(a.ID), c.Request.Method)
		if bindError == nil {
			response, requestParamLog = a.HttpHandle(c, param)
		} else {
			response = gin.H{"RetCode": 230, "Message": fmt.Sprintf("Bind params error [%v]", bindError)}
		}
		jsonpCallback := ""
		if responseJSONPEnable {
			jsonpCallback = c.DefaultQuery(responseJSONPCallbackQueryName, "")
		}
		addAccessControlAllowHeader(c)
		if jsonpCallback == "" {
			c.JSON(http.StatusOK, response)
		} else {
			c.Render(http.StatusOK, render.JsonpJSON{Callback: jsonpCallback, Data: response})
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
