package security

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
	ErrJwtInvalidTokenLookup  = errors.New("invalid jwt token lookup")
	ErrJwtNotFoundQueryToken  = errors.New("not found jwt token in query")
	ErrJwtNotFoundHeaderToken = errors.New("not found jwt token in header")
	ErrJwtInvalidToken        = errors.New("invalid jwt token")
	ErrJwtExpired             = errors.New("jwt token expired")
)

const (
	JwtContextKey        = "jwt_claims"
	JwtTokenContextKey   = "jwt_token"
	DefaultExpireTime    = 3600 // 1 hour
	DefaultRefreshTime   = 1800 // 30 minutes
	DefaultTokenLookup   = "header:Authorization"
	DefaultRefreshHeader = "X-New-Token"
	DefaultSigningMethod = "HS256"
)

// JwtConfig JWT 配置结构
type JwtConfig struct {
	PrivateKey       string `mapstructure:"private_key"`       // JWT 私钥
	ExpireTime       int64  `mapstructure:"expire_time"`       // 过期时间（秒）
	TokenLookup      string `mapstructure:"token_lookup"`      // Token 查找位置 "header:Authorization" or "query:token"
	AutoRefresh      bool   `mapstructure:"auto_refresh"`      // 是否自动续期
	RefreshThreshold int64  `mapstructure:"refresh_threshold"` // 续期阈值（秒），距离过期还剩多少时间时续期
	RefreshHeader    string `mapstructure:"refresh_header"`    // 返回新 token 的 header 名称
	SigningMethod    string `mapstructure:"signing_method"`    // 签名算法 HS256/HS384/HS512
}

// JwtValidator JWT 验证器
type JwtValidator struct {
	config         JwtConfig
	tokenLookupMap [][2]string
	signingMethod  jwt.SigningMethod
}

// Claims JWT Claims 结构
type Claims struct {
	UserID   string                 `json:"user_id,omitempty"`
	Username string                 `json:"username,omitempty"`
	Data     map[string]interface{} `json:"data,omitempty"`
	jwt.RegisteredClaims
}

// NewJwtValidator 创建 JWT 验证器
func NewJwtValidator() *JwtValidator {
	jv := &JwtValidator{}
	jv.loadConfig()
	jv.parseTokenLookup()
	return jv
}

// loadConfig 从配置文件加载 JWT 配置
func (j *JwtValidator) loadConfig() {
	// 设置默认值
	config.SetDefault("security.jwt.expire_time", DefaultExpireTime)
	config.SetDefault("security.jwt.token_lookup", DefaultTokenLookup)
	config.SetDefault("security.jwt.auto_refresh", false)
	config.SetDefault("security.jwt.refresh_threshold", DefaultRefreshTime)
	config.SetDefault("security.jwt.refresh_header", DefaultRefreshHeader)
	config.SetDefault("security.jwt.signing_method", DefaultSigningMethod)

	// 从配置加载
	j.config.PrivateKey = config.GetString("security.jwt.private_key")
	j.config.ExpireTime = int64(config.GetInt("security.jwt.expire_time"))
	j.config.TokenLookup = config.GetString("security.jwt.token_lookup")
	j.config.AutoRefresh = config.GetBool("security.jwt.auto_refresh")
	j.config.RefreshThreshold = int64(config.GetInt("security.jwt.refresh_threshold"))
	j.config.RefreshHeader = config.GetString("security.jwt.refresh_header")
	j.config.SigningMethod = config.GetString("security.jwt.signing_method")

	// 设置签名方法
	switch j.config.SigningMethod {
	case "HS384":
		j.signingMethod = jwt.SigningMethodHS384
	case "HS512":
		j.signingMethod = jwt.SigningMethodHS512
	default:
		j.signingMethod = jwt.SigningMethodHS256
	}

	logger.Debugf("JWT config loaded: expire_time=%ds, auto_refresh=%v, refresh_threshold=%ds",
		j.config.ExpireTime, j.config.AutoRefresh, j.config.RefreshThreshold)
}

// parseTokenLookup 解析 token 查找配置
func (j *JwtValidator) parseTokenLookup() {
	fields := strings.Split(j.config.TokenLookup, ",")
	j.tokenLookupMap = make([][2]string, 0, len(fields))
	for _, field := range fields {
		parts := strings.Split(strings.TrimSpace(field), ":")
		if len(parts) == 2 {
			j.tokenLookupMap = append(j.tokenLookupMap, [2]string{parts[0], parts[1]})
		}
	}
}

// Valid 验证 JWT Token，支持自动续期
func (j *JwtValidator) Valid(c *gin.Context) error {
	tokenString, err := j.extractToken(c)
	if err != nil {
		return err
	}

	// 解析 token
	claims, err := j.ParseToken(tokenString)
	if err != nil {
		return err
	}

	// 将 claims 和原始 token 存入 context
	c.Set(JwtContextKey, claims)
	c.Set(JwtTokenContextKey, tokenString)

	// 自动续期检查
	if j.config.AutoRefresh {
		j.checkAndRefresh(c, claims)
	}

	return nil
}

// extractToken 从请求中提取 token
func (j *JwtValidator) extractToken(c *gin.Context) (string, error) {
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

// ParseToken 解析 JWT Token
func (j *JwtValidator) ParseToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// 验证签名方法
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

// GenerateToken 生成 JWT Token
func (j *JwtValidator) GenerateToken(userID, username string, data map[string]interface{}) (string, error) {
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

// RefreshToken 刷新 Token
func (j *JwtValidator) RefreshToken(oldClaims *Claims) (string, error) {
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

// checkAndRefresh 检查并自动续期
func (j *JwtValidator) checkAndRefresh(c *gin.Context, claims *Claims) {
	if claims.ExpiresAt == nil {
		return
	}

	// 计算距离过期还剩多少时间
	timeToExpire := time.Until(claims.ExpiresAt.Time).Seconds()

	// 如果剩余时间小于续期阈值，则自动续期
	if timeToExpire > 0 && timeToExpire < float64(j.config.RefreshThreshold) {
		newToken, err := j.RefreshToken(claims)
		if err != nil {
			logger.Errorf("Failed to refresh JWT token: %v", err)
			return
		}

		// 将新 token 写入响应头
		c.Header(j.config.RefreshHeader, newToken)
		logger.Debugf("JWT token auto-refreshed for user: %s, remaining time: %.0fs", claims.UserID, timeToExpire)
	}
}

// tokenFromQuery 从 query 参数中获取 token
func tokenFromQuery(c *gin.Context, key string) (string, error) {
	token := c.Query(key)
	if token == "" {
		return "", ErrJwtNotFoundQueryToken
	}
	return token, nil
}

// tokenFromHeader 从 header 中获取 token
func tokenFromHeader(c *gin.Context, key string) (string, error) {
	token := c.Request.Header.Get(key)
	if token == "" {
		return "", ErrJwtNotFoundHeaderToken
	}

	// 去除 "Bearer " 前缀
	token = strings.TrimPrefix(token, "Bearer ")
	token = strings.TrimPrefix(token, "bearer ")
	return strings.TrimSpace(token), nil
}

// tokenFromCookie 从 cookie 中获取 token
func tokenFromCookie(c *gin.Context, key string) (string, error) {
	token, err := c.Cookie(key)
	if err != nil || token == "" {
		return "", ErrJwtNotFoundHeaderToken
	}
	return token, nil
}

// ================ 辅助函数 ================

// GenerateToken 全局函数：生成 JWT Token
func GenerateToken(userID, username string, data map[string]interface{}) (string, error) {
	if validators["jwt"] == nil {
		return "", errors.New("jwt validator not initialized")
	}
	jv := validators["jwt"].(*JwtValidator)
	return jv.GenerateToken(userID, username, data)
}

// GetClaims 从 gin.Context 中获取 JWT Claims
func GetClaims(c *gin.Context) (*Claims, bool) {
	claims, exists := c.Get(JwtContextKey)
	if !exists {
		return nil, false
	}
	jwtClaims, ok := claims.(*Claims)
	return jwtClaims, ok
}

// GetUserID 从 gin.Context 中获取用户 ID
func GetUserID(c *gin.Context) string {
	claims, ok := GetClaims(c)
	if !ok {
		return ""
	}
	return claims.UserID
}

// GetUsername 从 gin.Context 中获取用户名
func GetUsername(c *gin.Context) string {
	claims, ok := GetClaims(c)
	if !ok {
		return ""
	}
	return claims.Username
}
