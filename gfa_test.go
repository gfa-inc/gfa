package gfa

import (
	"github.com/gfa-inc/gfa/common/logger"
	"github.com/gfa-inc/gfa/core"
	"github.com/gin-gonic/gin"
	"testing"
)

type testController struct {
}

type query struct {
	ID int `form:"id" binding:"required"`
}

func (*testController) hello(c *gin.Context) {
	var q query
	if err := c.ShouldBindQuery(&q); err != nil {
		logger.TError(c.Copy(), err)
		_ = c.Error(core.NewParamErr(err))
		return
	}

	logger.TInfo(c.Copy(), "hello")
	core.OK(c, "hello")
}

func (tc *testController) Setup(r *gin.RouterGroup) {
	PermitRoute("/hello")
	r.GET("/hello", tc.hello)
}

func TestRun(t *testing.T) {
	Default()

	AddController(&testController{})

	Run()
}
