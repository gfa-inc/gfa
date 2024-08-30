package gfa

import (
	"context"
	"github.com/gfa-inc/gfa/common/db/mysqlx"
	"github.com/gfa-inc/gfa/common/logger"
	"github.com/gfa-inc/gfa/common/mq/kafkax"
	"github.com/gfa-inc/gfa/core"
	"github.com/gfa-inc/gfa/middlewares/security"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/segmentio/kafka-go"
	"go.uber.org/zap/zapcore"
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

	session := sessions.Default(c)
	session.Set(security.SessionKey, q)
	err := session.Save()
	if err != nil {
		logger.TError(c, err)
		_ = c.Error(err)
		return
	}

	logger.TInfo(c.Copy(), "hello")
	core.OK(c, "hello")
}

func (tc *testController) Setup(r *gin.RouterGroup) {
	mysqlx.Client.Exec("select 1")
	PermitRoute("/hello")
	r.GET("/hello", tc.hello)
}

func TestRun(t *testing.T) {
	logger.RegisterCore("json", logger.NewJsonCore(logger.NewJsonCoreWriter(func(b []byte) (int, error) {
		if kafkax.ProducerClient == nil {
			return 0, nil
		}

		msg := make([]byte, len(b))
		copy(msg, b)
		err := kafkax.ProducerClient.WriteMessages(context.Background(), kafka.Message{
			Value: msg,
		})
		if err != nil {
			return 0, err
		}

		return len(b), nil
	}), logger.ToLevelPtr(zapcore.InfoLevel), func(cfg *zapcore.EncoderConfig) {
		cfg.EncodeTime = zapcore.TimeEncoderOfLayout("2006-01-02T15:04:05Z0700")
		cfg.TimeKey = "timestamp"
		cfg.MessageKey = "message"
		cfg.CallerKey = "logger"
	}))

	Default()

	AddController(&testController{})

	Run()
}
