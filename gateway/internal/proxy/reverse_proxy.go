package proxy

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	url2 "net/url"
	"strings"

	"github.com/gin-gonic/gin"
)

// Put userIDKey at package level, outside any function
type contextKey string

const userIDKey contextKey = "userID"

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

		// Read userID from context and set header here, AFTER originalDirector ran
		if id, ok := req.Context().Value(userIDKey).(string); ok && id != "" {
			req.Header.Set("User-ID", id)
		}
	}

	return func(c *gin.Context) {
		if v, ok := c.Get("userID"); ok {
			if id, ok2 := v.(string); ok2 {
				// Set header directly so downstream always receives it.
				c.Request.Header.Set("User-ID", id)
				c.Request = c.Request.WithContext(
					context.WithValue(c.Request.Context(), userIDKey, id),
				)
			}
		}

		proxy.ServeHTTP(c.Writer, c.Request)
	}
}
