package gfa

import (
	"github.com/gfa-inc/gfa/common/logger"
	"github.com/gfa-inc/gfa/core"
	"github.com/gin-gonic/gin"
	"testing"
)

type testController struct {
}

func (*testController) hello(c *gin.Context) {
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
