package session

import (
	"errors"

	"github.com/gfa-inc/gfa/common/config"
	"github.com/gfa-inc/gfa/common/logger"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

var (
	ErrNotFoundSession = errors.New("not found session")
)

const (
	ContextKey        = "session_user"
	DefaultSessionKey = "user"
)

// Config session configuration structure
type Config struct {
	SessionKey string `mapstructure:"session_key"` // Session key to look up in session store
}

// Validator session validator
type Validator struct {
	config Config
}

// Default creates a session validator from config file
func Default() *Validator {
	cfg := loadConfig()
	v := New(cfg)
	return v
}

// New creates a session validator with custom config
func New(cfg Config) *Validator {
	if cfg.SessionKey == "" {
		cfg.SessionKey = DefaultSessionKey
	}

	v := &Validator{
		config: cfg,
	}
	return v
}

// loadConfig loads session configuration from config file
func loadConfig() Config {
	var cfg Config
	// Load from config
	err := config.UnmarshalKey("security.session", &cfg)
	if err != nil {
		logger.Debugf("No session config found, using defaults")
	}

	if cfg.SessionKey == "" {
		cfg.SessionKey = DefaultSessionKey
	}

	logger.Debugf("Session config loaded: session_key=%s", cfg.SessionKey)

	return cfg
}

// Valid validates session
func (v *Validator) Valid(c *gin.Context) error {
	session := sessions.Default(c)
	value := session.Get(v.config.SessionKey)
	if value == nil {
		return ErrNotFoundSession
	}

	// Store session value in context for easy access
	c.Set(ContextKey, value)

	return nil
}

// GetSession retrieves session value from gin.Context
func GetSession(c *gin.Context) (interface{}, bool) {
	value, exists := c.Get(ContextKey)
	return value, exists
}

// GetSessionString retrieves session value as string from gin.Context
func GetSessionString(c *gin.Context) (string, bool) {
	value, exists := GetSession(c)
	if !exists {
		return "", false
	}
	str, ok := value.(string)
	return str, ok
}

// Key Deprecated: Use Config.SessionKey instead
var Key = DefaultSessionKey

// NewSessionValidator Deprecated: Use Default instead
func NewSessionValidator() *Validator {
	return Default()
}

// Option defines option for session middleware
type Option func(*Config)

// WithSessionKey sets the session key to look up
func WithSessionKey(key string) Option {
	return func(c *Config) {
		c.SessionKey = key
	}
}

// Middleware creates session middleware
// Used to protect specific routes, requires request to have a valid session
// opts: optional configuration using WithXXX functions
// Returns the validator instance and the middleware handler
func Middleware(opts ...Option) (*Validator, gin.HandlerFunc) {
	// Build Config from options
	cfg := Config{
		SessionKey: DefaultSessionKey,
	}

	for _, opt := range opts {
		opt(&cfg)
	}

	// Create validator with config
	validator := New(cfg)

	handler := func(c *gin.Context) {
		err := validator.Valid(c)
		if err != nil {
			logger.Warnf("Session validation failed: %v, path: %s", err, c.FullPath())
			c.AbortWithStatusJSON(401, gin.H{
				"error": "unauthorized",
			})
			return
		}
		c.Next()
	}

	return validator, handler
}
