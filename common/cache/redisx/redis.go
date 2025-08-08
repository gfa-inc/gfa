package redisx

import (
	"context"
	"strings"

	"github.com/gfa-inc/gfa/common/config"
	"github.com/gfa-inc/gfa/common/logger"
	"github.com/redis/go-redis/v9"
	"github.com/samber/lo"
)

var (
	Client     redis.UniversalClient
	clientPool map[string]redis.UniversalClient
)

type Config struct {
	Name     string
	Addrs    []string
	Username string
	Password string
	Default  bool
}

func NewClient(option Config) (redis.UniversalClient, error) {
	client := redis.NewUniversalClient(&redis.UniversalOptions{
		Addrs:    option.Addrs,
		Username: option.Username,
		Password: option.Password,
	})
	err := client.Ping(context.Background()).Err()
	if err != nil {
		logger.Panic(err)
	}

	logger.Infof("Connecting to redis [%s] %s successfully", option.Name, strings.Join(option.Addrs, ", "))
	return client, nil
}

func Setup() {
	clientPool = make(map[string]redis.UniversalClient)

	if config.Get("redis") == nil {
		logger.Debug("No redis config found")
		return
	}

	configMap := make(map[string]Config)
	err := config.UnmarshalKey("redis", &configMap)
	if err != nil {
		logger.Panic(err)
	}

	logger.Infof("Starting to initialize redis client pool")
	for name, option := range configMap {
		option.Name = name
		client, err := NewClient(option)
		if err != nil {
			logger.Panic(err)
		}
		PutClient(name, client)

		if option.Default {
			Client = client
		}
	}

	logger.Infof("Redis client pool has been initialized with %d clients, clients: %s",
		len(clientPool), strings.Join(lo.Keys(clientPool), ", "))
}

func GetClient(name string) redis.UniversalClient {
	client, ok := clientPool[name]
	if !ok {
		logger.Panicf("Redis client %s not found", name)
	}
	return client
}

func PutClient(name string, client redis.UniversalClient) {
	clientPool[name] = client
}
