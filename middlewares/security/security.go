package security

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gfa-inc/gfa/common/config"
	"github.com/gfa-inc/gfa/common/logger"
	"github.com/gfa-inc/gfa/middlewares/security/apikey"
	"github.com/gfa-inc/gfa/middlewares/security/jwtx"
	"github.com/gfa-inc/gfa/middlewares/security/session"
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
	PermittedFlag               = "security_permitted"
	Type                        = "security"
	DefaultSessionValidatorName = "session"
	DefaultJWTValidatorName     = "jwt"
	DefaultApiKeyValidatorName  = "api_key"
)

var (
	ErrNoSessionValidator = errors.New("no default session validator configured")
)

var (
	matcher          *router.RequestMatcher
	validators       map[string]Validator
	customValidators map[string]Validator
	apiPrefix        string
)

func init() {
	validators = make(map[string]Validator)
	customValidators = make(map[string]Validator)
}

func WithValidator(name string, v Validator) {
	customValidators[name] = v
}

func Security() gin.HandlerFunc {
	matcher = router.NewRequestMatcher()

	// Get API prefix from config, default to base_path
	apiPrefix = config.GetString("security.api_prefix")
	if apiPrefix == "" {
		apiPrefix = config.GetString("server.base_path")
	}
	logger.Debugf("Security API prefix: %s", apiPrefix)

	if config.Get("security.session") != nil {
		validators[DefaultSessionValidatorName] = session.Default()
	}
	if config.Get("security.jwt") != nil {
		validators[DefaultJWTValidatorName] = jwtx.Default()
	}
	if config.Get("security.api_key") != nil {
		validators[DefaultApiKeyValidatorName] = apikey.Default()
	}

	for k, v := range customValidators {
		validators[k] = v
	}

	logger.Debugf("Enabled security validators: %s", strings.Join(lo.Keys(validators), ", "))
	logger.Info("Security middleware enabled")

	return func(c *gin.Context) {
		// Check if the request path matches the API prefix
		// If API prefix is set and the path doesn't match, skip security validation
		if apiPrefix != "" && !strings.HasPrefix(c.Request.URL.Path, apiPrefix) {
			c.Next()
			return
		}

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

func SetSession(c *gin.Context, value any) error {
	v, ok := validators[DefaultSessionValidatorName]
	if !ok {
		return ErrNoSessionValidator
	}

	sessionValidator := v.(*session.Validator)
	sessionValidator.Set(c, value)
	return nil
}
