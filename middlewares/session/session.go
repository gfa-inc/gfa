package session

import (
	"net/http"

	"github.com/boj/redistore"
	"github.com/gfa-inc/gfa/common/config"
	"github.com/gfa-inc/gfa/common/logger"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/redis"
	"github.com/gin-gonic/gin"
)

var DefaultTimeout = 86400

type Config struct {
	PrivateKey string
	MaxAge     int
	Domain     string
	Redis      RedisConfig
}

type RedisConfig struct {
	MaxIdleConnSize int
	Network         string
	Addrs           []string
	Password        string
}

func Session() gin.HandlerFunc {
	option := Config{
		MaxAge: DefaultTimeout,
		Redis: RedisConfig{
			MaxIdleConnSize: 10,
			Network:         "tcp",
		},
	}
	err := config.UnmarshalKey("session", &option)
	if err != nil {
		logger.Panic(err)
	}

	if len(option.Redis.Addrs) == 0 {
		logger.Panic("No session config found")
	}
	newRedisStore, err := redis.NewStore(option.Redis.MaxIdleConnSize, option.Redis.Network, option.Redis.Addrs[0],
		option.Redis.Password, []byte(option.PrivateKey))
	if err != nil {
		logger.Panic(err)
	}

	newRedisStore.Options(sessions.Options{
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   option.MaxAge,
		Domain:   option.Domain,
	})

	_, store := redis.GetRedisStore(newRedisStore)
	store.SetMaxLength(1024 * 1024)
	store.SetKeyPrefix(config.GetString("name"))
	store.SetSerializer(redistore.JSONSerializer{})

	logger.Info("Session middleware enabled")
	return sessions.Sessions("_SESSIONID", newRedisStore)
}

func Enabled() bool {
	return config.Get("session") != nil
}
