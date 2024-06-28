package middlewares

import (
	"github.com/gfa-inc/gfa/common/config"
	"github.com/gfa-inc/gfa/common/logger"
	"github.com/gin-contrib/requestid"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func Requestid() gin.HandlerFunc {
	config.SetDefault("requestid.header_key", "X-Request-ID")
	headerKey := config.GetString("requestid.header_key")
	logger.Infof("Use requestid middleware, header key: %s", headerKey)
	return requestid.New(
		requestid.WithCustomHeaderStrKey(requestid.HeaderStrKey(headerKey)),
		requestid.WithGenerator(func() string {
			return uuid.NewString()
		}),
		requestid.WithHandler(func(c *gin.Context, requestID string) {
			c.Set("traceID", requestID)
		}))
}
