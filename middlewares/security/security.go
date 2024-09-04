package security

import (
	"github.com/duke-git/lancet/v2/maputil"
	"github.com/gfa-inc/gfa/common/config"
	"github.com/gfa-inc/gfa/common/logger"
	"github.com/gfa-inc/gfa/utils/router"
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
)

type Validator interface {
	Valid(c *gin.Context) error
}

const (
	PermittedFlag = "permitted"
	Type          = "security"
)

var (
	validators map[string]Validator
	matcher    *router.Matcher
)

func init() {
	validators = make(map[string]Validator)
}

func newSessionValidator() *SessionValidator {
	return &SessionValidator{}
}

func newJwtValidator() *JwtValidator {
	return &JwtValidator{}
}

func newApiKeyValidator() *ApiKeyValidator {
	config.SetDefault("security.api_key.header_key", "X-Api-Key")
	headerKey := config.GetString("security.api_key.header_key")
	return NewApiKeyValidator(headerKey)
}

func Security() gin.HandlerFunc {
	matcher = router.New()

	if config.Get("security.session") != nil {
		validators["session"] = newSessionValidator()
	}
	if config.Get("security.jwt") != nil {
		validators["jwt"] = newJwtValidator()
	}
	if config.Get("security.api_key") != nil {
		validators["api_key"] = newApiKeyValidator()
	}

	logger.Debugf("Enabled security validators: %s", strings.Join(maputil.Keys(validators), ", "))
	logger.Info("Security middleware enabled")

	return func(c *gin.Context) {
		if matcher.Match(c.FullPath()) {
			c.Set(PermittedFlag, true)
			c.Next()
			return
		}

		if validators == nil {
			c.Next()
			return
		}

		for k, v := range validators {
			if v.Valid(c) == nil {
				c.Set(Type, k)
				c.Next()
				return
			}
		}

		logger.Errorf("Unauthorized access: %s", c.FullPath())

		c.AbortWithStatus(http.StatusUnauthorized)
	}
}

func Enabled() bool {
	return config.Get("security") != nil
}

func PermitRoute(route string) {
	if matcher == nil {
		logger.Debug("Security middleware is not enabled")
		return
	}

	basePath := config.GetString("server.base_path")
	if basePath != "" && !strings.HasPrefix(route, basePath) {
		route = strings.Join([]string{basePath, route}, "")
	}

	matcher.AddRoute(route)
	logger.Debugf("Permit route %s", route)
}

func PermitRoutes(routes []string) {
	for _, route := range routes {
		PermitRoute(route)
	}
}

func IsPermitted(c *gin.Context) bool {
	return c.GetBool(PermittedFlag)
}
