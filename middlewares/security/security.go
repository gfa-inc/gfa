package security

import (
	"net/http"
	"strings"

	"github.com/gfa-inc/gfa/common/config"
	"github.com/gfa-inc/gfa/common/logger"
	"github.com/gfa-inc/gfa/utils/router"
	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
)

type Validator interface {
	Valid(c *gin.Context) error
}

type ValidatorWrapper struct {
	f func(c *gin.Context) error
}

func (vw *ValidatorWrapper) Valid(c *gin.Context) error {
	return vw.f(c)
}

func NewValidator(f func(c *gin.Context) error) *ValidatorWrapper {
	return &ValidatorWrapper{f: f}
}

const (
	PermittedFlag = "security_permitted"
	Type          = "security"
)

var (
	matcher          *router.RequestMatcher
	validators       map[string]Validator
	customValidators map[string]Validator
)

func init() {
	validators = make(map[string]Validator)
	customValidators = make(map[string]Validator)
}

func newApiKeyValidator() *ApiKeyValidator {
	config.SetDefault("security.api_key.header_key", "X-Api-Key")
	headerKey := config.GetString("security.api_key.header_key")
	return NewApiKeyValidator(headerKey, config.GetString("security.api_key.lookup"))
}

func WithValidator(name string, v Validator) {
	customValidators[name] = v
}

func Security() gin.HandlerFunc {
	matcher = router.NewRequestMatcher()

	if config.Get("security.session") != nil {
		validators["session"] = NewSessionValidator()
	}
	if config.Get("security.jwt") != nil {
		validators["jwt"] = NewJwtValidator()
	}
	if config.Get("security.api_key") != nil {
		validators["api_key"] = newApiKeyValidator()
	}

	for k, v := range customValidators {
		validators[k] = v
	}

	logger.Debugf("Enabled security validators: %s", strings.Join(lo.Keys(validators), ", "))
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
