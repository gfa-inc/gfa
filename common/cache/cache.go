package cache

import (
	"context"
	"errors"
	"time"

	"github.com/bytedance/sonic"
	"github.com/gfa-inc/gfa/common/cache/hash"
	"github.com/gfa-inc/gfa/common/cache/redisx"
	"github.com/gfa-inc/gfa/common/logger"
	"github.com/redis/go-redis/v9"
)

func Setup() {
	redisx.Setup()
}

func Key[T any](ctx context.Context, prefix string, value T) (string, error) {
	key, err := hash.Hash(ctx, value)
	if err != nil {
		return "", err
	}

	return prefix + key, nil
}

func RSet[T any](ctx context.Context, client redis.UniversalClient, key string, value T, expiration time.Duration) error {
	if client == nil {
		client = redisx.Client
	}

	b, err := sonic.Marshal(value)
	if err != nil {
		logger.TError(ctx, err)
		return err
	}

	err = client.Set(ctx, key, b, expiration).Err()
	if err != nil {
		logger.TError(ctx, err)
		return err
	}

	return nil
}

func RGet[T any](ctx context.Context, client redis.UniversalClient, key string) (T, error) {
	if client == nil {
		client = redisx.Client
	}

	var value T

	data, err := client.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			logger.TWarnf(ctx, "key %s not found", key)
			return value, nil
		} else {
			logger.TError(ctx, err)
			return value, err
		}
	}

	err = sonic.Unmarshal(data, &value)
	if err != nil {
		logger.TError(ctx, err)
		return value, err
	}

	return value, nil
}
