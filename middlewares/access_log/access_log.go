package access_log

import (
	"time"

	"github.com/gfa-inc/gfa/common/logger"
	"github.com/gfa-inc/gfa/utils/router"
	"github.com/gin-gonic/gin"
)

var (
	whitelistMatcher *router.RequestMatcher
	ClientIPKey      = "clientIP"
	LatencyKey       = "latency"
)

func AccessLog() gin.HandlerFunc {
	whitelistMatcher = router.NewRequestMatcher()

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

		if whitelistMatcher.Match(path, method) {
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

	whitelistMatcher.AddRoute(route, method)
	logger.Debugf("AccessLog middleware permit route %s", route)
}

func PermitRoutes(routes [][]any) {
	if whitelistMatcher == nil {
		logger.Debug("access_log middleware is not enabled")
		return
	}

	whitelistMatcher.AddRoutes(routes)
}
