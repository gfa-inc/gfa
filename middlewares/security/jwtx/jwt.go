package jwtx

import (
	"errors"
	"strings"
	"time"

	"github.com/gfa-inc/gfa/common/config"
	"github.com/gfa-inc/gfa/common/logger"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrJwtNotFoundQueryToken  = errors.New("not found jwt token in query")
	ErrJwtNotFoundHeaderToken = errors.New("not found jwt token in header")
	ErrJwtInvalidToken        = errors.New("invalid jwt token")
	ErrJwtExpired             = errors.New("jwt token expired")
)

const (
	JwtContextKey           = "jwt_claims"
	JwtTokenContextKey      = "jwt_token"
	DefaultExpireTime       = 3600 // 1 hour
	DefaultRefreshThreshold = 1800 // 30 minutes
	DefaultTokenLookup      = "header:Authorization"
	DefaultRefreshHeader    = "X-New-Token"
	DefaultSigningMethod    = "HS256"
)

// Config JWT configuration structure
type Config struct {
	PrivateKey       string `mapstructure:"private_key"`       // JWT private key
	ExpireTime       int64  `mapstructure:"expire_time"`       // Token expiration time in seconds
	TokenLookup      string `mapstructure:"token_lookup"`      // Token lookup location "header:Authorization" or "query:token"
	AutoRefresh      bool   `mapstructure:"auto_refresh"`      // Enable auto token refresh
	RefreshThreshold int64  `mapstructure:"refresh_threshold"` // Refresh threshold in seconds, time remaining before expiration to trigger refresh
	RefreshHeader    string `mapstructure:"refresh_header"`    // Response header name for new token
	SigningMethod    string `mapstructure:"signing_method"`    // Signing algorithm HS256/HS384/HS512
}

// Validator JWT validator
type Validator struct {
	config         Config
	tokenLookupMap [][2]string
	signingMethod  jwt.SigningMethod
}

// Claims JWT claims structure
type Claims struct {
	UserID   string                 `json:"user_id,omitempty"`
	Username string                 `json:"username,omitempty"`
	Data     map[string]interface{} `json:"data,omitempty"`
	jwt.RegisteredClaims
}

// Default creates a JWT validator from config file
func Default() *Validator {
	cfg := loadConfig()
	jv := New(cfg)
	return jv
}

func New(cfg Config) *Validator {
	if cfg.ExpireTime == 0 {
		cfg.ExpireTime = DefaultExpireTime
	}
	if cfg.TokenLookup == "" {
		cfg.TokenLookup = DefaultTokenLookup
	}
	if cfg.RefreshThreshold == 0 {
		cfg.RefreshThreshold = DefaultRefreshThreshold
	}
	if cfg.RefreshHeader == "" {
		cfg.RefreshHeader = DefaultRefreshHeader
	}
	if cfg.SigningMethod == "" {
		cfg.SigningMethod = DefaultSigningMethod
	}

	jv := &Validator{
		config: cfg,
	}
	jv.parseTokenLookup()
	jv.parseSigningMethod()
	return jv
}

// loadConfig loads JWT configuration from config file
func loadConfig() Config {
	var cfg Config
	// Load from config
	err := config.UnmarshalKey("security.jwt", &cfg)
	if err != nil {
		logger.Panicf("Failed to load JWT config: %v", err)
		return cfg
	}

	logger.Debugf("JWT config loaded: expire_time=%ds, auto_refresh=%v, refresh_threshold=%ds",
		cfg.ExpireTime, cfg.AutoRefresh, cfg.RefreshThreshold)

	return cfg
}

// parseTokenLookup parses token lookup configuration
func (j *Validator) parseTokenLookup() {
	fields := strings.Split(j.config.TokenLookup, ",")
	j.tokenLookupMap = make([][2]string, 0, len(fields))
	for _, field := range fields {
		parts := strings.Split(strings.TrimSpace(field), ":")
		if len(parts) == 2 {
			j.tokenLookupMap = append(j.tokenLookupMap, [2]string{parts[0], parts[1]})
		}
	}
}

func (j *Validator) parseSigningMethod() {
	switch j.config.SigningMethod {
	case "HS384":
		j.signingMethod = jwt.SigningMethodHS384
	case "HS512":
		j.signingMethod = jwt.SigningMethodHS512
	default:
		j.signingMethod = jwt.SigningMethodHS256
	}
}

// Valid validates JWT token with auto-refresh support
func (j *Validator) Valid(c *gin.Context) error {
	tokenString, err := j.extractToken(c)
	if err != nil {
		return err
	}

	// Parse token
	claims, err := j.ParseToken(tokenString)
	if err != nil {
		return err
	}

	// Store claims and original token in context
	c.Set(JwtContextKey, claims)
	c.Set(JwtTokenContextKey, tokenString)

	// Check for auto-refresh
	if j.config.AutoRefresh {
		j.checkAndRefresh(c, claims)
	}

	return nil
}

// extractToken extracts token from request
func (j *Validator) extractToken(c *gin.Context) (string, error) {
	for _, parts := range j.tokenLookupMap {
		lookup, key := parts[0], parts[1]

		var token string
		var err error

		switch lookup {
		case "query":
			token, err = tokenFromQuery(c, key)
		case "header":
			token, err = tokenFromHeader(c, key)
		case "cookie":
			token, err = tokenFromCookie(c, key)
		default:
			token, err = tokenFromHeader(c, key)
		}

		if err == nil && token != "" {
			return token, nil
		}
	}
	return "", ErrJwtNotFoundHeaderToken
}

// ParseToken parses JWT token
func (j *Validator) ParseToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if token.Method != j.signingMethod {
			return nil, errors.New("invalid signing method")
		}
		return []byte(j.config.PrivateKey), nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrJwtExpired
		}
		return nil, ErrJwtInvalidToken
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, ErrJwtInvalidToken
}

// GenerateToken generates a JWT token
func (j *Validator) GenerateToken(userID, username string, data map[string]interface{}) (string, error) {
	now := time.Now()
	claims := &Claims{
		UserID:   userID,
		Username: username,
		Data:     data,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(j.config.ExpireTime) * time.Second)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(j.signingMethod, claims)
	return token.SignedString([]byte(j.config.PrivateKey))
}

// RefreshToken refreshes an existing token
func (j *Validator) RefreshToken(oldClaims *Claims) (string, error) {
	now := time.Now()
	newClaims := &Claims{
		UserID:   oldClaims.UserID,
		Username: oldClaims.Username,
		Data:     oldClaims.Data,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(j.config.ExpireTime) * time.Second)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(j.signingMethod, newClaims)
	return token.SignedString([]byte(j.config.PrivateKey))
}

// checkAndRefresh checks and performs auto-refresh if needed
func (j *Validator) checkAndRefresh(c *gin.Context, claims *Claims) {
	if claims.ExpiresAt == nil {
		return
	}

	// Calculate time remaining before expiration
	timeToExpire := time.Until(claims.ExpiresAt.Time).Seconds()

	// Auto-refresh if remaining time is less than threshold
	if timeToExpire > 0 && timeToExpire < float64(j.config.RefreshThreshold) {
		newToken, err := j.RefreshToken(claims)
		if err != nil {
			logger.Errorf("Failed to refresh JWT token: %v", err)
			return
		}

		// Write new token to response header
		c.Header(j.config.RefreshHeader, newToken)
		logger.Debugf("JWT token auto-refreshed for user: %s, remaining time: %.0fs", claims.UserID, timeToExpire)
	}
}

// tokenFromQuery extracts token from query parameters
func tokenFromQuery(c *gin.Context, key string) (string, error) {
	token := c.Query(key)
	if token == "" {
		return "", ErrJwtNotFoundQueryToken
	}
	return token, nil
}

// tokenFromHeader extracts token from request header
func tokenFromHeader(c *gin.Context, key string) (string, error) {
	token := c.Request.Header.Get(key)
	if token == "" {
		return "", ErrJwtNotFoundHeaderToken
	}

	// Remove "Bearer " prefix
	token = strings.TrimPrefix(token, "Bearer ")
	token = strings.TrimPrefix(token, "bearer ")
	return strings.TrimSpace(token), nil
}

// tokenFromCookie extracts token from cookie
func tokenFromCookie(c *gin.Context, key string) (string, error) {
	token, err := c.Cookie(key)
	if err != nil || token == "" {
		return "", ErrJwtNotFoundHeaderToken
	}
	return token, nil
}

// GetClaims retrieves JWT claims from gin.Context
func GetClaims(c *gin.Context) (*Claims, bool) {
	claims, exists := c.Get(JwtContextKey)
	if !exists {
		return nil, false
	}
	jwtClaims, ok := claims.(*Claims)
	return jwtClaims, ok
}

// GetUserID retrieves user ID from gin.Context
func GetUserID(c *gin.Context) string {
	claims, ok := GetClaims(c)
	if !ok {
		return ""
	}
	return claims.UserID
}

// GetUsername retrieves username from gin.Context
func GetUsername(c *gin.Context) string {
	claims, ok := GetClaims(c)
	if !ok {
		return ""
	}
	return claims.Username
}

// Option defines option for JWT middleware
type Option func(*Config)

// WithPrivateKey sets the JWT private key
func WithPrivateKey(key string) Option {
	return func(c *Config) {
		c.PrivateKey = key
	}
}

// WithExpireTime sets the token expiration time in seconds
func WithExpireTime(expire int64) Option {
	return func(c *Config) {
		c.ExpireTime = expire
	}
}

// WithTokenLookup sets the token lookup location (e.g., "header:Authorization" or "query:token")
func WithTokenLookup(lookup string) Option {
	return func(c *Config) {
		c.TokenLookup = lookup
	}
}

// WithAutoRefresh enables or disables auto token refresh
func WithAutoRefresh(autoRefresh bool) Option {
	return func(c *Config) {
		c.AutoRefresh = autoRefresh
	}
}

// WithRefreshThreshold sets the refresh threshold in seconds
func WithRefreshThreshold(threshold int64) Option {
	return func(c *Config) {
		c.RefreshThreshold = threshold
	}
}

// WithRefreshHeader sets the response header name for new token
func WithRefreshHeader(header string) Option {
	return func(c *Config) {
		c.RefreshHeader = header
	}
}

// WithSigningMethod sets the signing algorithm (HS256/HS384/HS512)
func WithSigningMethod(method string) Option {
	return func(c *Config) {
		c.SigningMethod = method
	}
}

// Middleware creates JWT middleware
// Used to protect specific routes, requires request to have a valid JWT token
// opts: optional configuration using WithXXX functions
// Returns the validator instance and the middleware handler
func Middleware(opts ...Option) (*Validator, gin.HandlerFunc) {
	// Build Config from options
	cfg := Config{
		ExpireTime:       DefaultExpireTime,
		TokenLookup:      DefaultTokenLookup,
		RefreshThreshold: DefaultRefreshThreshold,
		RefreshHeader:    DefaultRefreshHeader,
		SigningMethod:    DefaultSigningMethod,
	}

	for _, opt := range opts {
		opt(&cfg)
	}

	// Create validator with config
	validator := New(cfg)

	handler := func(c *gin.Context) {
		err := validator.Valid(c)
		if err != nil {
			logger.Warnf("JWT validation failed: %v, path: %s", err, c.FullPath())
			c.AbortWithStatusJSON(401, gin.H{
				"error": "unauthorized",
			})
			return
		}
		c.Next()
	}

	return validator, handler
}
