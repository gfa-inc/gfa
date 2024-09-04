package config

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestSetup(t *testing.T) {
	Setup(WithPath("../../"))
	assert.NotEmpty(t, GetString("server.addr"))
	assert.NotEmpty(t, Get("server.addr"))
	assert.NotEmpty(t, Get("logger"))
}
