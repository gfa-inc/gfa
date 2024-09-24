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
	PermittedFlag = "security_permitted"
	Type          = "security"
)

var (
	validators map[string]Validator
	matcher    *router.RequestMatcher
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
	matcher = router.NewRequestMatcher()

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
		if matcher.Match(c.FullPath(), c.Request.Method) {
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

func PermitRoute(route string, method any) {
	if matcher == nil {
		logger.Debug("Security middleware is not enabled")
		return
	}

	matcher.AddRoute(route, method)
	logger.Debugf("Security middleware permit route %s", route)
}

func PermitRoutes(routes [][]any) {
	for _, route := range routes {
		PermitRoute(route[0].(string), route[1])
	}
}

func IsPermitted(c *gin.Context) bool {
	return c.GetBool(PermittedFlag)
}
