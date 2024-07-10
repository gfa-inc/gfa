package security

import (
	"github.com/duke-git/lancet/v2/maputil"
	"github.com/gfa-inc/gfa/common/config"
	"github.com/gfa-inc/gfa/common/logger"
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
)

type Validator interface {
	Valid(c *gin.Context) error
}

var (
	validators      map[string]Validator
	permittedRoutes map[string]struct{}
)

func init() {
	validators = make(map[string]Validator)
	permittedRoutes = make(map[string]struct{})
}

func newSessionValidator() *SessionValidator {
	return &SessionValidator{}
}

func newJwtValidator() *JwtValidator {
	return &JwtValidator{}
}

func newApiKeyValidator() *ApiKeyValidator {
	config.SetDefault("security.api_key.header_key", "X-API-KEY")
	headerKey := config.GetString("security.api_key.header_key")
	return NewApiKeyValidator(headerKey)
}

func Security() gin.HandlerFunc {
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
	logger.Info("Use security middleware")

	return func(c *gin.Context) {
		if _, ok := permittedRoutes[c.FullPath()]; ok {
			c.Next()
			return
		}

		if validators == nil {
			c.Next()
			return
		}

		for _, v := range validators {
			if v.Valid(c) == nil {
				c.Next()
				return
			}
		}

		logger.Infof("Unauthorized access: %s", c.FullPath())
		c.AbortWithStatus(http.StatusUnauthorized)
	}
}

func Enabled() bool {
	return config.Get("security") != nil
}

func PermitRoute(route string) {
	basePath := config.GetString("server.base_path")
	if basePath != "" && !strings.HasPrefix(route, basePath) {
		route = strings.Join([]string{basePath, route}, "")
	}

	logger.Debugf("Permit route %s", route)
	permittedRoutes[route] = struct{}{}
}

func PermitRoutes(routes []string) {
	for _, route := range routes {
		PermitRoute(route)
	}
}
