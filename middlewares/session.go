package middlewares

import (
	"github.com/gfa-inc/gfa/common/config"
	"github.com/gfa-inc/gfa/common/logger"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/redis"
	"github.com/gin-gonic/gin"
	"net/http"
)

var DefaultTimeout = 86400

func Session() gin.HandlerFunc {
	addrs := config.GetStringSlice("session.redis.addrs")
	if len(addrs) == 0 {
		logger.Panic("No session config found")
	}
	newRedisStore, err := redis.NewStore(10, "tcp", addrs[0],
		config.GetString("session.redis.password"), []byte(config.GetString("session.private_key")))
	if err != nil {
		logger.Panic(err)
	}

	config.SetDefault("session.max_age", DefaultTimeout)
	newRedisStore.Options(sessions.Options{
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   config.GetInt("session.max_age"),
		//Domain:   config.GetString("server.domain"),
	})

	logger.Info("Use session middleware")
	return sessions.Sessions("_SESSIONID", newRedisStore)
}

func SessionEnabled() bool {
	return config.Get("session") != nil
}
