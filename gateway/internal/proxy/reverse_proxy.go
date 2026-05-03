package proxy

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	url2 "net/url"
	"strings"

	"github.com/gin-gonic/gin"
)

type Resolver interface {
	Resolve(service string) (string, error)
}

func ReverseProxy(fromPrefix, toPrefix, target string) gin.HandlerFunc {
	url, err := url2.Parse(target)
	if err != nil {
		panic(fmt.Sprintf("invalid service url %s: %v", target, err))
	}

	proxy := httputil.NewSingleHostReverseProxy(url)

	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)

		if strings.HasPrefix(req.URL.Path, fromPrefix) {
			req.URL.Path = toPrefix + strings.TrimPrefix(req.URL.Path, fromPrefix)
		}

		req.Header.Set("Forwarded-Host", req.Header.Get("Host"))
		req.Host = url.Host
	}

	return func(c *gin.Context) {
		// Propagate authenticated user info to downstream services via headers.
		if v, ok := c.Get("userID"); ok {
			if id, ok2 := v.(uint64); ok2 {
				c.Request.Header.Set("User-ID", fmt.Sprintf("%d", id))
			}
		}

		if v, ok := c.Get("role"); ok {
			if role, ok2 := v.(string); ok2 {
				c.Request.Header.Set("User-Role", role)
			}
		}
		
		proxy.ServeHTTP(c.Writer, c.Request)
	}
}
