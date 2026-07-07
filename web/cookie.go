package web

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

const defaultCookieMaxAge = 157680000 //five years

//RemoveCookie 清除Cookie
func RemoveCookie(c *gin.Context, name string) {
	WriteCookie2(c, name, "", "/", true, 0)
}

//ReadCookie 读取Cookie
func ReadCookie(c *gin.Context, name string) (string, error) {
	var value string
	cookie, err := c.Request.Cookie(name)
	if err == nil {
		value = cookie.Value
	}
	return value, err
}

//WriteCookie 记录Cookie,使用缺省 Path:/ 缺省 HttpOnly:true 缺省 MaxAge:157680000 五年
func WriteCookie(c *gin.Context, name string, value string) {
	WriteCookie2(c, name, value, "/", true, defaultCookieMaxAge)
}

//WriteCookie2 记录Cookie
func WriteCookie2(c *gin.Context, name string, value string, path string, httpOnly bool, maxAge int) {
	cookie := &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     path,
		HttpOnly: httpOnly,
		MaxAge:   maxAge,
	}
	http.SetCookie(c.Writer, cookie)
}
