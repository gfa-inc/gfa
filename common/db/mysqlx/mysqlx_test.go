package mysqlx

import (
	"testing"

	"github.com/gfa-inc/gfa/common/config"
	"github.com/gfa-inc/gfa/common/logger"
	"github.com/stretchr/testify/assert"
)

func TestNewMysqlClient(t *testing.T) {
	logger.Setup()
	client, err := NewClient(Config{
		DNS:   "gfa:123456@tcp(127.0.0.1:3306)/gfa?charset=utf8mb4&parseTime=True&loc=Local",
		Level: "debug",
	})
	client.Debug().Exec("select 1")
	assert.Nil(t, err)
	assert.NotNil(t, client)
}

func TestSetup(t *testing.T) {
	config.Setup(config.WithPath("../../../../"))
	logger.Setup()
	Setup()
	assert.NotNil(t, Client)
	assert.NotNil(t, GetClient("default"))
}
