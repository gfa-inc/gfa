package s3x

import (
	"github.com/gfa-inc/gfa/common/config"
	"github.com/gfa-inc/gfa/common/logger"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSetup(t *testing.T) {
	config.Setup(config.WithPath("../../../../"))
	logger.Setup()
	Setup()
	assert.NotNil(t, Client)
	assert.NotNil(t, GetClient("default"))
}
