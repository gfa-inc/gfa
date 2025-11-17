package once_token

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gfa-inc/gfa/common/cache/redisx"
	"github.com/gfa-inc/gfa/common/config"
	"github.com/gfa-inc/gfa/common/logger"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

var (
	ErrOnceTokenNotFound       = errors.New("once token not found")
	ErrOnceTokenInvalid        = errors.New("once token invalid")
	ErrOnceTokenExpired        = errors.New("once token expired")
	ErrOnceTokenPathMismatch   = errors.New("once token path mismatch")
	ErrOnceTokenRedisNotConfig = errors.New("redis client not configured")
)

const (
	OnceTokenContextKey    = "once_token"
	DefaultOnceTokenExpire = 300 // 5 minutes
	DefaultOnceTokenLookup = "header:X-Once-Token"
	DefaultOnceTokenPrefix = "once_token:"
)

// OnceTokenConfig once token configuration
type OnceTokenConfig struct {
	Expire      int64  `mapstructure:"expire"`       // Expiration time in seconds
	TokenLookup string `mapstructure:"token_lookup"` // Token lookup location "header:X-Once-Token" or "query:token"
	Prefix      string `mapstructure:"prefix"`       // Redis key prefix
}

// OnceTokenValidator once token validator
type OnceTokenValidator struct {
	config         OnceTokenConfig
	tokenLookupMap [][2]string
	redisClient    redis.UniversalClient
}

// NewOnceTokenValidator creates a once token validator
func NewOnceTokenValidator() *OnceTokenValidator {
	otv := &OnceTokenValidator{}
	otv.loadConfig()
	otv.parseTokenLookup()

	// Use default Redis client
	if redisx.Client == nil {
		logger.Panic(ErrOnceTokenRedisNotConfig)
	}
	otv.redisClient = redisx.Client

	return otv
}

// NewOnceTokenValidatorWithConfig creates a once token validator with custom config
func NewOnceTokenValidatorWithConfig(cfg OnceTokenConfig, redisClient ...redis.UniversalClient) *OnceTokenValidator {
	otv := &OnceTokenValidator{
		config: cfg,
	}

	// Apply defaults if not set
	if otv.config.Expire <= 0 {
		otv.config.Expire = DefaultOnceTokenExpire
	}
	if otv.config.TokenLookup == "" {
		otv.config.TokenLookup = DefaultOnceTokenLookup
	}
	if otv.config.Prefix == "" {
		otv.config.Prefix = DefaultOnceTokenPrefix
	}

	otv.parseTokenLookup()

	// Use custom Redis client or default
	if len(redisClient) > 0 && redisClient[0] != nil {
		otv.redisClient = redisClient[0]
	} else {
		if redisx.Client == nil {
			logger.Panic(ErrOnceTokenRedisNotConfig)
		}
		otv.redisClient = redisx.Client
	}

	logger.Debugf("OnceToken validator created with custom config: expire=%ds, prefix=%s",
		otv.config.Expire, otv.config.Prefix)

	return otv
}

// loadConfig loads configuration from config file
func (ot *OnceTokenValidator) loadConfig() {
	// Set default values
	config.SetDefault("security.once_token.expire", DefaultOnceTokenExpire)
	config.SetDefault("security.once_token.token_lookup", DefaultOnceTokenLookup)
	config.SetDefault("security.once_token.prefix", DefaultOnceTokenPrefix)

	// Load from config
	ot.config.Expire = int64(config.GetInt("security.once_token.expire"))
	ot.config.TokenLookup = config.GetString("security.once_token.token_lookup")
	ot.config.Prefix = config.GetString("security.once_token.prefix")

	logger.Debugf("OnceToken config loaded: expire=%ds, prefix=%s",
		ot.config.Expire, ot.config.Prefix)
}

// parseTokenLookup parses token lookup configuration
func (ot *OnceTokenValidator) parseTokenLookup() {
	fields := strings.Split(ot.config.TokenLookup, ",")
	ot.tokenLookupMap = make([][2]string, 0, len(fields))
	for _, field := range fields {
		parts := strings.Split(strings.TrimSpace(field), ":")
		if len(parts) == 2 {
			ot.tokenLookupMap = append(ot.tokenLookupMap, [2]string{parts[0], parts[1]})
		}
	}
}

// Valid validates once token
func (ot *OnceTokenValidator) Valid(c *gin.Context) error {
	tokenString, err := ot.extractToken(c)
	if err != nil {
		return err
	}

	// Validate and consume token
	path, err := ot.ValidateAndConsumeToken(c.Request.Context(), tokenString, c.FullPath())
	if err != nil {
		return err
	}

	// Store token and path in context
	c.Set(OnceTokenContextKey, tokenString)
	c.Set("once_token_path", path)

	return nil
}

// extractToken extracts token from request
func (ot *OnceTokenValidator) extractToken(c *gin.Context) (string, error) {
	for _, parts := range ot.tokenLookupMap {
		lookup, key := parts[0], parts[1]

		var token string
		var err error

		switch lookup {
		case "query":
			token = c.Query(key)
		case "header":
			token = c.GetHeader(key)
		case "cookie":
			token, err = c.Cookie(key)
		default:
			token = c.GetHeader(key)
		}

		if err == nil && token != "" {
			return token, nil
		}
	}
	return "", ErrOnceTokenNotFound
}

// GenerateToken generates a once token
func (ot *OnceTokenValidator) GenerateToken(ctx context.Context, path string, expire ...time.Duration) (string, error) {
	// Generate random token
	token, err := ot.generateRandomToken()
	if err != nil {
		return "", err
	}

	// Set expiration time
	expireTime := time.Duration(ot.config.Expire) * time.Second
	if len(expire) > 0 && expire[0] > 0 {
		expireTime = expire[0]
	}

	// Store to Redis
	key := ot.config.Prefix + token
	err = ot.redisClient.Set(ctx, key, path, expireTime).Err()
	if err != nil {
		logger.Errorf("Failed to store once token to redis: %v", err)
		return "", err
	}

	logger.Debugf("Generated once token for path: %s, expire: %v", path, expireTime)
	return token, nil
}

// ValidateAndConsumeToken validates and consumes once token
func (ot *OnceTokenValidator) ValidateAndConsumeToken(ctx context.Context, token string, currentPath string) (string, error) {
	if token == "" {
		return "", ErrOnceTokenInvalid
	}

	key := ot.config.Prefix + token

	// Get token's associated path from Redis
	storedPath, err := ot.redisClient.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return "", ErrOnceTokenExpired
		}
		logger.Errorf("Failed to get once token from redis: %v", err)
		return "", err
	}

	// Validate path if specified
	if storedPath != "" && currentPath != "" && storedPath != currentPath {
		logger.Warnf("Once token path mismatch: expected=%s, actual=%s", storedPath, currentPath)
		return "", ErrOnceTokenPathMismatch
	}

	// Delete token (consume it)
	err = ot.redisClient.Del(ctx, key).Err()
	if err != nil {
		logger.Errorf("Failed to delete once token from redis: %v", err)
		return "", err
	}

	logger.Debugf("Validated and consumed once token for path: %s", storedPath)
	return storedPath, nil
}

// generateRandomToken generates a random token using UUID32 format
func (ot *OnceTokenValidator) generateRandomToken() (string, error) {
	// Generate UUID and remove hyphens to get 32-character hex string
	token := strings.ReplaceAll(uuid.New().String(), "-", "")
	return token, nil
}

// GetOnceToken gets once token from gin.Context
func GetOnceToken(c *gin.Context) (string, bool) {
	token, exists := c.Get(OnceTokenContextKey)
	if !exists {
		return "", false
	}
	tokenStr, ok := token.(string)
	return tokenStr, ok
}

var globalOnceTokenValidator *OnceTokenValidator

// InitOnceToken initializes global once token validator (optional, can also be auto-initialized via security config)
func InitOnceToken() {
	if globalOnceTokenValidator == nil {
		globalOnceTokenValidator = NewOnceTokenValidator()
	}
}

// GenerateOnceToken generates a once token (global function)
// path: specifies the path the token can only be used for, empty string means no path restriction
// expire: optional expiration time, uses default from config if not provided
func GenerateOnceToken(ctx context.Context, path string, expire ...time.Duration) (string, error) {
	if globalOnceTokenValidator == nil {
		InitOnceToken()
	}
	return globalOnceTokenValidator.GenerateToken(ctx, path, expire...)
}

// ValidateOnceToken validates and consumes once token (global function)
func ValidateOnceToken(ctx context.Context, token string, currentPath string) (string, error) {
	if globalOnceTokenValidator == nil {
		InitOnceToken()
	}
	return globalOnceTokenValidator.ValidateAndConsumeToken(ctx, token, currentPath)
}

// OnceTokenOption defines option for OnceTokenMiddleware
type OnceTokenOption func(*OnceTokenConfig)

// OnceTokenMiddlewareConfig holds middleware configuration
type OnceTokenMiddlewareConfig struct {
	expire      *int64
	tokenLookup *string
	prefix      *string
	redisClient redis.UniversalClient
}

// WithExpire sets the token expiration time in seconds
func WithExpire(expire int64) OnceTokenOption {
	return func(c *OnceTokenConfig) {
		c.Expire = expire
	}
}

// WithTokenLookup sets the token lookup location (e.g., "header:X-Once-Token" or "query:token")
func WithTokenLookup(lookup string) OnceTokenOption {
	return func(c *OnceTokenConfig) {
		c.TokenLookup = lookup
	}
}

// WithPrefix sets the Redis key prefix
func WithPrefix(prefix string) OnceTokenOption {
	return func(c *OnceTokenConfig) {
		c.Prefix = prefix
	}
}

// OnceTokenMiddleware creates once token middleware
// Used to protect specific routes, requires request to have a valid once token
// opts: optional configuration using WithXXX functions
// Returns the validator instance and the middleware handler
func OnceTokenMiddleware(opts ...OnceTokenOption) (*OnceTokenValidator, gin.HandlerFunc) {
	var validator *OnceTokenValidator

	// Build OnceTokenConfig from middleware config
	cfg := OnceTokenConfig{
		Expire:      DefaultOnceTokenExpire,
		TokenLookup: DefaultOnceTokenLookup,
		Prefix:      DefaultOnceTokenPrefix,
	}

	for _, opt := range opts {
		opt(&cfg)
	}

	// Create validator with config
	validator = NewOnceTokenValidatorWithConfig(cfg)

	handler := func(c *gin.Context) {
		err := validator.Valid(c)
		if err != nil {
			logger.Warnf("Once token validation failed: %v, path: %s", err, c.FullPath())
			c.AbortWithStatusJSON(401, gin.H{
				"error": fmt.Sprintf("once token validation failed: %v", err),
			})
			return
		}
		c.Next()
	}

	return validator, handler
}
