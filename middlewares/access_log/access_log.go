package access_log

import (
	"fmt"
	"github.com/gfa-inc/gfa/common/logger"
	"github.com/gfa-inc/gfa/utils/router"
	"github.com/gin-gonic/gin"
	"time"
)

var (
	Matcher     *router.Matcher
	ClientIPKey = "clientIP"
	LatencyKey  = "latency"
)

func AccessLog() gin.HandlerFunc {
	Matcher = router.New()

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

		key := fmt.Sprintf("%s#%s", path, method)
		if Matcher.Match(key) {
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
