package access_log

import (
	"fmt"
	"github.com/gfa-inc/gfa/common/config"
	"github.com/gfa-inc/gfa/common/logger"
	"github.com/gfa-inc/gfa/utils/http_method"
	"github.com/gfa-inc/gfa/utils/router"
	"github.com/gin-gonic/gin"
	"strings"
	"time"
)

var (
	whitelistMatcher *router.Matcher
	ClientIPKey      = "clientIP"
	LatencyKey       = "latency"
)

func AccessLog() gin.HandlerFunc {
	whitelistMatcher = router.New()

	logger.AddContextKey(ClientIPKey)
	logger.AddContextKey(LatencyKey)

	logger.Info("Access log middleware enabled")
	return func(c *gin.Context) {
		start := time.Now()
		// request method
		method := c.Request.Method
		// request path
		path := c.Request.URL.Path
		// client ip
		clientIP := c.ClientIP()

		c.Set(ClientIPKey, clientIP)

		c.Next()

		key := genKey(path, method)
		if whitelistMatcher.Match(key) {
			return
		}

		// request latency
		latency := time.Now().Sub(start).Milliseconds()
		c.Set(LatencyKey, latency)
		// response status
		status := c.Writer.Status()
		// error message
		errorMessage := c.Errors.ByType(gin.ErrorTypePrivate).String()

		logger.TInfof(c, "[%d] %s %s %dms %s", status, method, path, latency, errorMessage)
	}
}

func PermitRoute(route string, method any) {
	if whitelistMatcher == nil {
		logger.Debug("access_log middleware is not enabled")
		return
	}

	basePath := config.GetString("server.base_path")
	if basePath != "" && !strings.HasPrefix(route, basePath) {
		route = strings.Join([]string{basePath, route}, "")
	}

	normalizedMethods, err := http_method.Normalize(method)
	if err != nil {
		logger.Error(err)
		return
	}

	for _, m := range normalizedMethods {
		whitelistMatcher.AddRoute(genKey(route, m))
	}
}

func genKey(route string, method string) string {
	return fmt.Sprintf("%s#%s", route, method)
}
