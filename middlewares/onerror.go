package middlewares

import (
	"github.com/gfa-inc/gfa/common/logger"
	"github.com/gfa-inc/gfa/core"
	"github.com/gin-gonic/gin"
	"net/http"
)

func OnError() gin.HandlerFunc {
	logger.Info("Use onerror middleware")
	return func(c *gin.Context) {
		c.Next()

		// return if no error
		if len(c.Errors) == 0 {
			return
		}

		for _, err := range c.Errors {
			switch e := err.Err.(type) {
			case *core.ParameterError:
				_ = c.AbortWithError(http.StatusBadRequest, e)
			case *core.BizError:
				c.AbortWithStatusJSON(http.StatusOK, core.NewFailedResponse(e.Code, e.Message))
			default:
				_ = c.AbortWithError(http.StatusInternalServerError, err)
			}
		}
	}
}
