package middlewares

import (
	"github.com/gfa-inc/gfa/common/logger"
	"github.com/gfa-inc/gfa/core"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
)

func OnError() gin.HandlerFunc {
	logger.Info("Onerror middleware enabled")
	return func(c *gin.Context) {
		c.Next()

		// return if no error
		if len(c.Errors) == 0 {
			return
		}

		logger.Error(c.Errors.String())
		for _, err := range c.Errors {
			switch e := err.Err.(type) {
			case *core.ParamErr:
				c.AbortWithStatusJSON(http.StatusBadRequest, core.NewFailedResponse(c, strconv.Itoa(http.StatusBadRequest), e.Error()))
			case *core.BizErr:
				c.AbortWithStatusJSON(http.StatusOK, core.NewFailedResponse(c, e.Code, e.Message))
			default:
				_ = c.AbortWithError(http.StatusInternalServerError, err)
			}
			return
		}
	}
}
