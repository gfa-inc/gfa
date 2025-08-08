package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSetup(t *testing.T) {
	Setup(WithPath("../../"))
	assert.NotEmpty(t, GetString("server.addr"))
	assert.NotEmpty(t, Get("server.addr"))
	assert.NotEmpty(t, Get("logger"))
}
