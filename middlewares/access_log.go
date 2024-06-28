package middlewares

import (
	"github.com/gfa-inc/gfa/common/logger"
	"github.com/gin-contrib/requestid"
	"github.com/gin-gonic/gin"
	"time"
)

func AccessLog() gin.HandlerFunc {
	logger.Info("Use access log middleware")
	return func(c *gin.Context) {
		start := time.Now()
		// request path
		path := c.Request.URL.Path
		// client ip
		clientIP := c.ClientIP()
		// request method
		method := c.Request.Method
		// request id
		requestID := requestid.Get(c)

		c.Next()

		// request latency
		latency := time.Now().Sub(start).Milliseconds()
		// response status
		status := c.Writer.Status()
		// error message
		errorMessage := c.Errors.ByType(gin.ErrorTypePrivate).String()
		logger.Infof("%s [%d] %s %s %dms %s %s", clientIP, status, method, path, latency, requestID, errorMessage)
	}
}
