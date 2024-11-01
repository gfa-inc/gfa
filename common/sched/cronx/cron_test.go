package cronx

import (
	"github.com/gfa-inc/gfa/common/config"
	"github.com/gfa-inc/gfa/common/logger"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSetup(t *testing.T) {
	config.Setup(config.WithPath("../../../"))
	logger.Setup()
	Setup()

	ch := make(chan int64)
	id, err := C.AddFunc("0 * * * *", func() {
		logger.Info("every hour")
	})
	assert.Nil(t, err)

	id, err = C.AddFunc("*/1 * * * * *", func() {
		logger.Info("every 1 seconds")
		ch <- 1
	})
	assert.Nil(t, err)

	<-ch
	C.Remove(id)
}
