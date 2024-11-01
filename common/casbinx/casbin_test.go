package casbinx

import (
	"github.com/gfa-inc/gfa/common/config"
	"github.com/gfa-inc/gfa/common/db"
	"github.com/gfa-inc/gfa/common/logger"
	"github.com/spf13/cast"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSetup(t *testing.T) {
	config.Setup(config.WithPath("../../../.."))
	logger.Setup()
	db.Setup()

	Setup()

	_, err := Enforcer.HasRoleForUser(cast.ToString(1), "admin")
	assert.Nil(t, err)
}
