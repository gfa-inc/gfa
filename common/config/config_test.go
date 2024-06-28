package config

import (
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSetup(t *testing.T) {
	Setup(WithPath("../../../"))
	assert.NotEmpty(t, viper.GetString("server.addr"))
	assert.NotEmpty(t, GetString("server.addr"))
	assert.Equal(t, "127.0.0.1:8080", GetString("server.addr"))
}
