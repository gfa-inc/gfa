package hash

import (
	"context"
	"testing"

	"github.com/gfa-inc/gfa/common/config"
	"github.com/gfa-inc/gfa/common/logger"
	"github.com/stretchr/testify/assert"
)

func TestHash(t *testing.T) {
	config.Setup(config.WithPath("../../../"))
	logger.Setup()

	hash, err := Hash(context.Background(), map[string]any{
		"name": "test",
	})
	assert.Nil(t, err)
	assert.NotEmpty(t, hash)

	hash1, err := Hash(context.Background(), map[string]any{
		"name": "test",
	})
	assert.Nil(t, err)
	assert.NotEmpty(t, hash1)
	assert.Equal(t, hash, hash1)
}
