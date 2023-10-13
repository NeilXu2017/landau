package data

import (
	"bytes"
	"crypto/sha1"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/NeilXu2017/landau/log"
)

type (
	// HTTPHelperOptionFunc 参数设置
	HTTPHelperOptionFunc func(*HTTPHelper) error
	// HTTPMethod 请求Web的方法，支持GET,POST
	HTTPMethod string
	// HTTPHelper 访问 http 服务包装类
	HTTPHelper struct {
		method                     HTTPMethod
		url                        string
		requestParams              map[string]interface{} // map 结构请求参数
		requestRawObject           interface{}            // 非map结构请求参数
		postBody                   string                 //Post方法，自定义body内容
		timeout                    int
		userAgent                  string
		contentType                string
		delegatedHTTPRequest       *http.Request
		requestHead                map[string]string
		logger                     string
		publicKey                  string // 密钥对签名方式访问
		privateKey                 string
		publicKeyParaName          string
		signatureParaName          string
		logRequest                 func(interface{}) string
		logResponse                func(interface{}) string
		showLogResponseAll         bool
		showLogResponseSummarySize int
		Response                   *http.Response
		debugSignature             bool
		logSignature               string
		debugResponseHeaderField   []string
		insecureSkipVerify         bool
	}
	_HttpCookieJar struct {
		cookies []*http.Cookie
		url     *url.URL
	}
)

const (
	// HTTPGet GET方法
	HTTPGet HTTPMethod = "GET"
	// HTTPPost POST方法
	HTTPPost HTTPMethod = "POST"
	//HTTPPut PUT 方法
	HTTPPut HTTPMethod = "PUT"
	//HTTPDelete DELETE 方法
	HTTPDelete HTTPMethod = "DELETE"
)

// NewHTTPHelper 构造HTTPHelper实例
func NewHTTPHelper(options ...HTTPHelperOptionFunc) (*HTTPHelper, error) {
	c := &HTTPHelper{
		method:                     HTTPPost,
		requestParams:              make(map[string]interface{}),
		postBody:                   "",
		timeout:                    5,
		userAgent:                  "Mozilla/5.0 (Windows; U; Windows NT 6.0; en-US; rv:1.9.0.5) Gecko/2008120122 Firefox/3.0.5",
		contentType:                "application/json",
		requestHead:                make(map[string]string),
		logger:                     "main",
		publicKey:                  "",
		privateKey:                 "",
		publicKeyParaName:          "PublicKey",
		signatureParaName:          "Signature",
		showLogResponseAll:         false,
		showLogResponseSummarySize: 2048,
		debugSignature:             false,
		debugResponseHeaderField:   []string{},
	}
	for _, option := range options {
		if err := option(c); err != nil {
			return nil, err
		}
	}
	return c, nil
}

// CallHTTPService 调用HTTP接口
func CallHTTPService(method HTTPMethod, url string, params map[string]interface{}, responseObject interface{}) error {
	httpHelper, err := NewHTTPHelper(
		SetHTTPUrl(url),
		SetHTTPMethod(method),
		SetHTTPRequestParams(params),
	)
	if err != nil {
		return err
	}
	return httpHelper.Call2(responseObject)
}

// CallHTTPService2 json 方式调用
func CallHTTPService2(url string, rawObject interface{}, responseObject interface{}) error {
	httpHelper, err := NewHTTPHelper(
		SetHTTPUrl(url),
		SetHTTPRequestRawObject(rawObject),
	)
	if err != nil {
		return err
	}
	return httpHelper.Call2(responseObject)
}

func (c *HTTPHelper) defaultLogResponse(response interface{}) string {
	msg := ""
	if v, ok := response.(fmt.Stringer); ok {
		msg = v.String()
	} else {
		msg = fmt.Sprintf("%v", response)
	}
	if utf8.RuneCountInString(msg) <= c.showLogResponseSummarySize {
		return msg
	}
	responseRune := []rune(msg)
	shownLen := len(responseRune) / 2
	if shownLen > 64 {
		shownLen = 64
	}
	return fmt.Sprintf("%s......%s", string(responseRune[0:shownLen]), string(responseRune[len(responseRune)-shownLen:]))
}

// SetHTTPUrl 设置 HTTP 请求 url 地址
func SetHTTPUrl(url string) HTTPHelperOptionFunc {
	return func(c *HTTPHelper) error {
		c.url = url
		return nil
	}
}

// SetHTTPMethod 设置 HTTP 请求method
func SetHTTPMethod(method HTTPMethod) HTTPHelperOptionFunc {
	return func(c *HTTPHelper) error {
		c.method = method
		return nil
	}
}

// SetHTTPRequestParams 设置 HTTP 请求参数
func SetHTTPRequestParams(params map[string]interface{}) HTTPHelperOptionFunc {
	return func(c *HTTPHelper) error {
		c.requestParams = params
		return nil
	}
}

// SetHTTPRequestRawObject 设置 HTTP 请求参数
func SetHTTPRequestRawObject(rawObject interface{}) HTTPHelperOptionFunc {
	return func(c *HTTPHelper) error {
		c.requestRawObject = rawObject
		return nil
	}
}

// SetHTTPTimeout  设置 HTTP 请求超时时间，单位秒
func SetHTTPTimeout(timeout int) HTTPHelperOptionFunc {
	return func(c *HTTPHelper) error {
		c.timeout = timeout
		return nil
	}
}

// SetHTTPUserAgent  设置 HTTP UserAgent 信息
func SetHTTPUserAgent(userAgent string) HTTPHelperOptionFunc {
	return func(c *HTTPHelper) error {
		c.userAgent = userAgent
		return nil
	}
}

// SetHTTPContentType  设置 HTTP ContentType 信息
func SetHTTPContentType(contentType string) HTTPHelperOptionFunc {
	return func(c *HTTPHelper) error {
		c.contentType = contentType
		return nil
	}
}

// SetHTTPLogCategory  设置 HTTP Logger 名称
func SetHTTPLogCategory(logCategory string) HTTPHelperOptionFunc {
	return func(c *HTTPHelper) error {
		c.logger = logCategory
		return nil
	}
}

// SetHTTPResponseLogger  设置 HTTP Response Logger 名称
func SetHTTPResponseLogger(responseLogger func(interface{}) string) HTTPHelperOptionFunc {
	return func(c *HTTPHelper) error {
		c.logResponse = responseLogger
		return nil
	}
}

// SetHTTPRequestLogger  设置 HTTP Request Logger 名称
func SetHTTPRequestLogger(requestLogger func(interface{}) string) HTTPHelperOptionFunc {
	return func(c *HTTPHelper) error {
		c.logRequest = requestLogger
		return nil
	}
}

// SetHTTPPublicKey  设置 HTTP 签名 PublicKey
func SetHTTPPublicKey(publicKey string) HTTPHelperOptionFunc {
	return func(c *HTTPHelper) error {
		c.publicKey = publicKey
		return nil
	}
}

// SetHTTPPrivateKey  设置 HTTP 签名 PublicKey
func SetHTTPPrivateKey(privateKey string) HTTPHelperOptionFunc {
	return func(c *HTTPHelper) error {
		c.privateKey = privateKey
		return nil
	}
}

// SetHTTPPublicKeyParaName  设置 HTTP 签名参数名：公钥参数
func SetHTTPPublicKeyParaName(publicKeyParaName string) HTTPHelperOptionFunc {
	return func(c *HTTPHelper) error {
		c.publicKeyParaName = publicKeyParaName
		return nil
	}
}

// SetHTTPSignatureParaName  设置 HTTP 签名参数名：签名参数
func SetHTTPSignatureParaName(signatureParaName string) HTTPHelperOptionFunc {
	return func(c *HTTPHelper) error {
		c.signatureParaName = signatureParaName
		return nil
	}
}

// SetHTTPShowLogResponseAll  设置 HTTP Response Log All 参数
func SetHTTPShowLogResponseAll(showLogResponseAll bool) HTTPHelperOptionFunc {
	return func(c *HTTPHelper) error {
		c.showLogResponseAll = showLogResponseAll
		return nil
	}
}

// SetHTTPShowLogResponseSummarySize  设置 HTTP Response Log Summary size 参数
func SetHTTPShowLogResponseSummarySize(showLogResponseSummarySize int) HTTPHelperOptionFunc {
	return func(c *HTTPHelper) error {
		c.showLogResponseSummarySize = showLogResponseSummarySize
		return nil
	}
}

// SetHTTPRequestHead 设置 HTTP requestHead 参数
func SetHTTPRequestHead(requestHead map[string]string) HTTPHelperOptionFunc {
	return func(c *HTTPHelper) error {
		c.requestHead = requestHead
		return nil
	}
}

// AppendHTTPRequestHeader 设置 HTTP requestHead 参数
func AppendHTTPRequestHeader(name, value string) HTTPHelperOptionFunc {
	return func(c *HTTPHelper) error {
		if c.requestHead == nil {
			c.requestHead = make(map[string]string)
		}
		c.requestHead[name] = value
		return nil
	}
}

// SetHTTPDelegatedHTTPRequest 设置 HTTP delegatedHTTPRequest 参数
func SetHTTPDelegatedHTTPRequest(delegatedHTTPRequest *http.Request) HTTPHelperOptionFunc {
	return func(c *HTTPHelper) error {
		c.delegatedHTTPRequest = delegatedHTTPRequest
		return nil
	}
}

// SetHTTPPostBody 设置 HTTP postBody 参数
func SetHTTPPostBody(postBody string) HTTPHelperOptionFunc {
	return func(c *HTTPHelper) error {
		c.postBody = postBody
		return nil
	}
}

// AppendHTTPDebugResponseHeader 设置 HTTP Debug Response Head 参数
func AppendHTTPDebugResponseHeader(name string) HTTPHelperOptionFunc {
	return func(c *HTTPHelper) error {
		if c.debugResponseHeaderField == nil {
			c.debugResponseHeaderField = []string{}
		}
		c.debugResponseHeaderField = append(c.debugResponseHeaderField, name)
		return nil
	}
}

func SetHTTPInsecureSkipVerify(insecureSkipVerify bool) HTTPHelperOptionFunc {
	return func(c *HTTPHelper) error {
		c.insecureSkipVerify = insecureSkipVerify
		return nil
	}
}

// SetHTTPDebugSignature 设置 HTTP debugSignature 参数
func SetHTTPDebugSignature(debugSignature bool) HTTPHelperOptionFunc {
	return func(c *HTTPHelper) error {
		c.debugSignature = debugSignature
		return nil
	}
}

func (c *_HttpCookieJar) SetCookies(u *url.URL, cookies []*http.Cookie) {
	c.cookies = cookies
	c.url = u
}

func (c *_HttpCookieJar) Cookies(u *url.URL) []*http.Cookie {
	if u == c.url {
		return c.cookies
	}
	return c.cookies
}

func (c *HTTPHelper) _prepareRequest() (string, string, io.Reader, string) {
	reqURL, reqMethod, postBody, signature, debugSignatureStr := c.url, "", "", "", ""
	if c.publicKey != "" && c.privateKey != "" && c.requestParams != nil { //需要签名
		c.requestParams[c.publicKeyParaName] = c.publicKey
		signature, debugSignatureStr = getSha1Sign(c.requestParams, c.privateKey)
		if c.debugSignature {
			c.logSignature = debugSignatureStr
		}
	}
	strQuery := getEncodedQueryString(c.requestParams)
	switch c.method {
	case HTTPGet:
		reqMethod = "GET"
		if strQuery != "" {
			symlinkChar := "?"
			if strings.Index(reqURL, "?") > 0 {
				symlinkChar = "&"
			}
			reqURL = fmt.Sprintf("%s%s%s", reqURL, symlinkChar, strQuery)
			if signature != "" {
				reqURL = fmt.Sprintf("%s&%s=%s", reqURL, c.signatureParaName, signature)
			}
		}
	case HTTPPost, HTTPDelete, HTTPPut:
		reqMethod = string(c.method)
		if c.postBody != "" {
			postBody = c.postBody
		} else {
			if c.requestRawObject == nil {
				if signature != "" {
					c.requestParams[c.signatureParaName] = signature
					b, _ := json.Marshal(c.requestParams)
					postBody = string(b)
				} else {
					postBody = strQuery
				}
			} else {
				b, _ := json.Marshal(&(c.requestRawObject))
				postBody = string(b)
			}
		}
	}
	requestLoggerMsg := postBody
	if c.logRequest != nil {
		var v interface{}
		if c.requestRawObject != nil {
			v = c.requestRawObject
		} else {
			v = c.requestParams
		}
		requestLoggerMsg = c.logRequest(v)
	}
	return reqURL, reqMethod, strings.NewReader(postBody), requestLoggerMsg
}

// Call 调用 HTTP 服务
func (c *HTTPHelper) Call() (string, error) {
	start := time.Now()
	client := &http.Client{Timeout: time.Duration(c.timeout) * time.Second}
	if c.insecureSkipVerify {
		client.Transport = &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
		}
	}
	if c.delegatedHTTPRequest != nil {
		jar := &_HttpCookieJar{}
		jar.cookies = c.delegatedHTTPRequest.Cookies()
		client.Jar = jar
	}
	reqURL, reqMethod, bodyReader, requestLoggerMsg := c._prepareRequest()
	req, err := http.NewRequest(reqMethod, reqURL, bodyReader)
	if err != nil {
		log.Error2(c.logger, "[HTTP]\t[%s]\tURL:%s\t%s\tError:%v", time.Since(start), reqURL, requestLoggerMsg, err)
		return "", err
	}
	req.Header.Set("User-Agent", c.userAgent)
	if c.contentType != "" {
		req.Header.Set("Content-Type", c.contentType)
	}
	if c.requestHead != nil {
		for k, v := range c.requestHead {
			req.Header.Set(k, v)
		}
	}
	response, responseErr := client.Do(req)
	c.Response = response
	if responseErr != nil {
		log.Error2(c.logger, "[HTTP]\t[%s]\tURL:%s\t%s\tError:%v", time.Since(start), reqURL, requestLoggerMsg, responseErr)
		return "", responseErr
	}
	responseByteBody, readResponseErr := io.ReadAll(response.Body)
	if readResponseErr != nil {
		log.Error2(c.logger, "[HTTP]\t[%s]\tURL:%s\t%s\tError:%v", time.Since(start), reqURL, requestLoggerMsg, readResponseErr)
		return "", readResponseErr
	}
	responseBody := string(responseByteBody)
	responseLoggerMsg := ""
	if c.logResponse != nil {
		responseLoggerMsg = c.logResponse(responseBody)
	} else {
		responseLoggerMsg = c.defaultLogResponse(responseBody)
	}
	if len(c.debugResponseHeaderField) > 0 {
		for _, key := range c.debugResponseHeaderField {
			value := response.Header.Get(key)
			responseLoggerMsg = fmt.Sprintf("%s %s=%s", responseLoggerMsg, key, value)
		}
	}
	if log.IsEnableCategoryInfoLog("HTTP") {
		log.Info2(c.logger, "[HTTP]\t[%s]\tURL:%s\t%s\tResponse:%s", time.Since(start), reqURL, requestLoggerMsg, responseLoggerMsg)
		if c.debugSignature {
			log.Info2(c.logger, "[HTTP-Signature-Debug] [%s]", c.logSignature)
		}
	}
	return responseBody, nil
}

// Call2 调用 HTTP 服务
func (c *HTTPHelper) Call2(responseObject interface{}) error {
	response, err := c.Call()
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(response), &responseObject)
}

// Upload 上传
func (c *HTTPHelper) Upload(fileFieldName string, filePath string) (string, error) {
	start := time.Now()
	var err error
	responseBody := ""
	defer func() {
		if err == nil {
			responseLoggerMsg := ""
			if c.logResponse != nil {
				responseLoggerMsg = c.logResponse(responseBody)
			} else {
				responseLoggerMsg = c.defaultLogResponse(responseBody)
			}
			if log.IsEnableCategoryInfoLog("HTTP") {
				log.Info2(c.logger, "[HTTPUpload]\t[%s]\tURL:%s filePath:%s\tResponse:%s", time.Since(start), c.url, filePath, responseLoggerMsg)
			}
		} else {
			log.Error2(c.logger, "[HTTPUpload]\t[%s]\tURL:%s filePath:%s\tResponse:%stError:%v", time.Since(start), c.url, filePath, responseBody, err)
		}
	}()
	var file *os.File
	if file, err = os.Open(filePath); err != nil {
		return "", err
	}
	defer file.Close()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	var part io.Writer
	if part, err = writer.CreateFormFile(fileFieldName, filepath.Base(filePath)); err != nil {
		return "", err
	}
	if _, err = io.Copy(part, file); err != nil {
		return "", err
	}
	if c.requestParams != nil {
		for k, v := range c.requestParams {
			if vStr, ok := v.(fmt.Stringer); ok {
				_ = writer.WriteField(k, vStr.String())
			} else {
				_ = writer.WriteField(k, fmt.Sprintf("%v", v))
			}
		}
	}
	if err = writer.Close(); err != nil {
		return "", err
	}
	var r *http.Request
	if r, err = http.NewRequest("POST", c.url, body); err != nil {
		return "", err
	}
	r.Header.Set("Content-Type", writer.FormDataContentType())
	r.Header.Set("User-Agent", c.userAgent)
	if c.requestHead != nil {
		for k, v := range c.requestHead {
			r.Header.Set(k, v)
		}
	}
	client := &http.Client{}
	if c.insecureSkipVerify {
		client.Transport = &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			ForceAttemptHTTP2:     true,
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			TLSClientConfig:       &tls.Config{InsecureSkipVerify: true},
		}
	}
	if c.delegatedHTTPRequest != nil {
		jar := &_HttpCookieJar{}
		jar.cookies = c.delegatedHTTPRequest.Cookies()
		client.Jar = jar
	}
	var resp *http.Response
	if resp, err = client.Do(r); err != nil {
		return "", err
	}
	c.Response = resp
	defer resp.Body.Close()
	var byteBody []byte
	if byteBody, err = io.ReadAll(resp.Body); err != nil {
		return "", err
	}
	responseBody = string(byteBody)
	return responseBody, nil
}

// Upload2 上传
func (c *HTTPHelper) Upload2(fileFieldName string, filePath string, responseObject interface{}) error {
	response, err := c.Upload(fileFieldName, filePath)
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(response), &responseObject)
}

func getEncodedQueryString(params map[string]interface{}) string {
	strQuery := ""
	if params != nil {
		u := url.Values{}
		for k, v := range params {
			if vStr, ok := v.(fmt.Stringer); ok {
				u.Set(k, vStr.String())
			} else {
				u.Set(k, fmt.Sprintf("%v", v))
			}
		}
		strQuery = u.Encode()
	}
	return strQuery
}

func getSha1Sign(params map[string]interface{}, privateKey string) (string, string) {
	strQuery := ""
	var paramNames []string
	for k := range params {
		paramNames = append(paramNames, k)
	}
	sort.Strings(paramNames)
	for i, j := 0, len(paramNames); i < j; i++ {
		v := params[paramNames[i]]
		if vStr, ok := v.(fmt.Stringer); ok {
			strQuery += fmt.Sprintf("%s%s", paramNames[i], vStr.String())
		} else {
			strQuery += fmt.Sprintf("%s%v", paramNames[i], v)
		}
	}
	strQuery += privateKey
	h := sha1.New()
	h.Write([]byte(strQuery))
	bs := h.Sum(nil)
	return fmt.Sprintf("%x", bs), strQuery
}
