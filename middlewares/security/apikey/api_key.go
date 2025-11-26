package apikey

import (
	"errors"
	"strings"

	"github.com/gfa-inc/gfa/common/config"
	"github.com/gfa-inc/gfa/common/logger"
	"github.com/gin-gonic/gin"
)

var (
	ErrNotFoundApiKey        = errors.New("not found api key")
	ErrApiKeyValidateFailed  = errors.New("api key validation failed")
	ErrValidateHandlerNotSet = errors.New("validate handler not set")
)

const (
	ContextKey    = "api_key"
	DefaultLookup = "header:X-Api-Key"
)

// Config API key configuration structure
type Config struct {
	Lookup          string                                    `mapstructure:"lookup"` // API key lookup location "header:X-Api-Key" or "query:api_key"
	ValidateHandler func(c *gin.Context, apiKey string) error `mapstructure:"-"`      // Custom validation handler
}

// Validator API key validator
type Validator struct {
	config          Config
	lookupMap       [][2]string
	validateHandler func(c *gin.Context, apiKey string) error
}

// Default creates an API key validator from config file
func Default() *Validator {
	cfg := loadConfig()
	v := New(cfg)
	return v
}

// New creates an API key validator with custom config
func New(cfg Config) *Validator {
	if cfg.Lookup == "" {
		cfg.Lookup = DefaultLookup
	}

	v := &Validator{
		config:          cfg,
		validateHandler: cfg.ValidateHandler,
	}
	v.parseLookup()
	return v
}

// loadConfig loads API key configuration from config file
func loadConfig() Config {
	var cfg Config
	// Load from config
	err := config.UnmarshalKey("security.api_key", &cfg)
	if err != nil {
		logger.Debugf("No api_key config found, using defaults")
	}

	logger.Debugf("API key config loaded: lookup=%s", cfg.Lookup)

	return cfg
}

// parseLookup parses API key lookup configuration
func (v *Validator) parseLookup() {
	fields := strings.Split(v.config.Lookup, ",")
	v.lookupMap = make([][2]string, 0, len(fields))
	for _, field := range fields {
		parts := strings.Split(strings.TrimSpace(field), ":")
		if len(parts) == 2 {
			v.lookupMap = append(v.lookupMap, [2]string{parts[0], parts[1]})
		}
	}
}

// Valid validates API key
func (v *Validator) Valid(c *gin.Context) error {
	apiKey, err := v.extractApiKey(c)
	if err != nil {
		return err
	}

	// Store API key in context
	c.Set(ContextKey, apiKey)

	// Validate using custom handler if set
	if v.validateHandler != nil {
		return v.validateHandler(c, apiKey)
	}

	// No validation handler means we only check if key exists
	return nil
}

// extractApiKey extracts API key from request
func (v *Validator) extractApiKey(c *gin.Context) (string, error) {
	for _, parts := range v.lookupMap {
		lookup, key := parts[0], parts[1]

		var apiKey string
		var err error

		switch lookup {
		case "query":
			apiKey = c.Query(key)
		case "header":
			apiKey = c.GetHeader(key)
		case "cookie":
			apiKey, err = c.Cookie(key)
		default:
			apiKey = c.GetHeader(key)
		}

		if err == nil && apiKey != "" {
			return apiKey, nil
		}
	}
	return "", ErrNotFoundApiKey
}

// GetApiKey retrieves API key from gin.Context
func GetApiKey(c *gin.Context) (string, bool) {
	apiKey, exists := c.Get(ContextKey)
	if !exists {
		return "", false
	}
	key, ok := apiKey.(string)
	return key, ok
}

// ValidateHandler SetValidateHandler sets the global validation handler
var ValidateHandler func(c *gin.Context, apiKey string) error

// Option defines option for API key middleware
type Option func(*Config)

// WithLookup sets the API key lookup location (e.g., "header:X-Api-Key" or "query:api_key")
func WithLookup(lookup string) Option {
	return func(c *Config) {
		c.Lookup = lookup
	}
}

// WithValidateHandler sets custom validation handler
func WithValidateHandler(handler func(c *gin.Context, apiKey string) error) Option {
	return func(c *Config) {
		c.ValidateHandler = handler
	}
}

// Middleware creates API key middleware
// Used to protect specific routes, requires request to have a valid API key
// opts: optional configuration using WithXXX functions
// Returns the validator instance and the middleware handler
func Middleware(opts ...Option) (*Validator, gin.HandlerFunc) {
	// Build Config from options
	cfg := Config{
		Lookup: DefaultLookup,
	}

	// Apply global ValidateHandler if set (for backward compatibility)
	if ValidateHandler != nil {
		cfg.ValidateHandler = ValidateHandler
	}

	for _, opt := range opts {
		opt(&cfg)
	}

	// Create validator with config
	validator := New(cfg)

	handler := func(c *gin.Context) {
		err := validator.Valid(c)
		if err != nil {
			logger.Warnf("API key validation failed: %v, path: %s", err, c.FullPath())
			c.AbortWithStatusJSON(401, gin.H{
				"error": "unauthorized",
			})
			return
		}
		c.Next()
	}

	return validator, handler
}
