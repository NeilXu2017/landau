package api

import (
	"bytes"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"strings"

	"github.com/NeilXu2017/landau/data"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

const (
	requestRawParams = "_RequestParams4Parse"
)

var (
	r *regexp.Regexp
)

func init() {
	r = regexp.MustCompile(`[\[\.]\d{1,9}[\]]*$`)
}

func prepareRequestParam(c *gin.Context, isPostMethod bool) {
	if isPostMethod {
		body, err := io.ReadAll(c.Request.Body)
		if err == nil {
			if len(body) > 0 {
				c.Set(requestRawParams, body)
				c.Request.Body = io.NopCloser(bytes.NewBuffer(body))
			} else {
				c.Set(requestRawParams, []byte("{}"))
			}
		}
	}
}

func bindQuery(obj interface{}, c *gin.Context) error {
	values := mergeArray(c.Request.URL.Query())
	if err := mapForm(obj, values); err != nil {
		return err
	}
	return binding.Validator.ValidateStruct(obj)
}

func bindPost(obj interface{}, c *gin.Context, isBindingComplex bool) error {
	if cb, ok := c.Get(requestRawParams); ok {
		if cbb, ok := cb.([]byte); ok {
			if isBindingComplex {
				if err := data.JSONUnmarshal(cbb, obj); err == nil {
					return nil
				}
			} else {
				if err := binding.JSON.BindBody(cbb, obj); err == nil {
					return nil
				}
			}
			vs, e := url.ParseQuery(string(cbb))
			if e != nil {
				return e
			}
			values := mergeArray(vs)
			if err := mapForm(obj, values); err != nil {
				return err
			}
			return binding.Validator.ValidateStruct(obj)
		}
	}
	return fmt.Errorf("cannot retreive post parameters")
}

func mergeArray(u url.Values) map[string][]string {
	o := make(map[string][]string)
	for k, v := range u {
		key := k
		if strings.Index(k, "[") > 0 || strings.Index(k, ".") > 0 {
			key = r.ReplaceAllString(k, "")
		}
		var values []string
		for _, rawValue := range v {
			s := strings.TrimSpace(rawValue)
			if s != "" {
				values = append(values, s)
			}
		}
		if len(values) > 0 {
			o[key] = append(o[key], values...)
		}
	}
	return o
}

func isPostBindingComplex(url string, action string) bool {
	if _, ok := postBindingComplexURL[url]; ok {
		return true
	}
	if _, ok := postBindingComplexAction[action]; ok {
		return true
	}
	return false
}
