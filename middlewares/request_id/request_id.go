package request_id

import (
	"github.com/gfa-inc/gfa/common/config"
	"github.com/gfa-inc/gfa/common/logger"
	"github.com/gin-contrib/requestid"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

var ContextKey string

type Config struct {
	HeaderKey  string
	ContextKey string
}

func RequestID() gin.HandlerFunc {
	option := Config{
		HeaderKey:  "X-Request-ID",
		ContextKey: "traceID",
	}
	err := config.UnmarshalKey("requestid", &option)
	if err != nil {
		logger.Panic(err)
	}

	ContextKey = option.ContextKey

	logger.Infof("Use requestid middleware, header key: %s", option.HeaderKey)
	return requestid.New(
		requestid.WithCustomHeaderStrKey(requestid.HeaderStrKey(option.HeaderKey)),
		requestid.WithGenerator(func() string {
			return uuid.NewString()
		}),
		requestid.WithHandler(func(c *gin.Context, requestID string) {
			c.Set(option.ContextKey, requestID)
		}))
}
