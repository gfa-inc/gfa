package cache

import (
	"context"
	"github.com/gfa-inc/gfa/common/config"
	"github.com/gfa-inc/gfa/common/logger"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestKey(t *testing.T) {
	key, err := Key(context.Background(), "prefix:", map[string]string{
		"name": "test",
	})
	assert.Nil(t, err)
	assert.NotEmpty(t, key)

	key1, err := Key(context.Background(), "prefix:", map[string]string{
		"name": "test",
	})
	assert.Nil(t, err)
	assert.NotEmpty(t, key1)
	assert.Equal(t, key, key1)
}

func TestRSet(t *testing.T) {
	config.Setup(config.WithPath("../../"))
	logger.Setup()
	Setup()

	k := "key"
	s := "test"
	err := RSet(context.Background(), nil, k, s, 5*time.Second)
	assert.Nil(t, err)

	s1, err := RGet[string](context.Background(), nil, k)
	assert.Nil(t, err)
	assert.Equal(t, s, s1)
}
