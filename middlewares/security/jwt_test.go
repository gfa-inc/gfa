package security

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gfa-inc/gfa/common/config"
	"github.com/gfa-inc/gfa/common/logger"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
)

func setupTest() {
	// 初始化配置
	config.Setup()

	// 设置测试配置
	config.SetDefault("security.jwt.private_key", "test-secret-key")
	config.SetDefault("security.jwt.expire_time", 3600)
	config.SetDefault("security.jwt.token_lookup", "header:Authorization")
	config.SetDefault("security.jwt.auto_refresh", true)
	config.SetDefault("security.jwt.refresh_threshold", 1800)
	config.SetDefault("security.jwt.refresh_header", "X-New-Token")
	config.SetDefault("security.jwt.signing_method", "HS256")

	// 初始化日志
	logger.Setup()
}

func TestJwtValidator_GenerateToken(t *testing.T) {
	setupTest()

	jv := NewJwtValidator()

	// 生成 token
	token, err := jv.GenerateToken("user123", "john_doe", map[string]interface{}{
		"role":  "admin",
		"email": "john@example.com",
	})

	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	// 解析 token 验证内容
	claims, err := jv.ParseToken(token)
	assert.NoError(t, err)
	assert.Equal(t, "user123", claims.UserID)
	assert.Equal(t, "john_doe", claims.Username)
	assert.Equal(t, "admin", claims.Data["role"])
	assert.Equal(t, "john@example.com", claims.Data["email"])
}

func TestJwtValidator_ParseToken(t *testing.T) {
	setupTest()

	jv := NewJwtValidator()

	// 生成有效 token
	validToken, _ := jv.GenerateToken("user123", "john_doe", nil)

	// 测试解析有效 token
	claims, err := jv.ParseToken(validToken)
	assert.NoError(t, err)
	assert.NotNil(t, claims)
	assert.Equal(t, "user123", claims.UserID)

	// 测试解析无效 token
	_, err = jv.ParseToken("invalid.token.here")
	assert.Error(t, err)

	// 测试解析过期 token
	expiredToken := createExpiredToken(t, jv)
	_, err = jv.ParseToken(expiredToken)
	assert.Error(t, err)
	assert.Equal(t, ErrJwtExpired, err)
}

func TestJwtValidator_Valid(t *testing.T) {
	setupTest()

	jv := NewJwtValidator()
	gin.SetMode(gin.TestMode)

	// 生成有效 token
	token, _ := jv.GenerateToken("user123", "john_doe", nil)

	// 创建测试请求
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)
	c.Request.Header.Set("Authorization", "Bearer "+token)

	// 验证 token
	err := jv.Valid(c)
	assert.NoError(t, err)

	// 验证 claims 已存入 context
	claims, exists := GetClaims(c)
	assert.True(t, exists)
	assert.Equal(t, "user123", claims.UserID)
	assert.Equal(t, "john_doe", claims.Username)
}

func TestJwtValidator_Valid_NoToken(t *testing.T) {
	setupTest()

	jv := NewJwtValidator()
	gin.SetMode(gin.TestMode)

	// 创建没有 token 的请求
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)

	// 验证应该失败
	err := jv.Valid(c)
	assert.Error(t, err)
	assert.Equal(t, ErrJwtNotFoundHeaderToken, err)
}

func TestJwtValidator_RefreshToken(t *testing.T) {
	setupTest()

	jv := NewJwtValidator()

	// 生成原始 token
	originalToken, _ := jv.GenerateToken("user123", "john_doe", map[string]interface{}{
		"role": "admin",
	})

	// 解析原始 token
	originalClaims, _ := jv.ParseToken(originalToken)

	// 等待一秒确保时间戳不同
	time.Sleep(1 * time.Second)

	// 刷新 token
	newToken, err := jv.RefreshToken(originalClaims)
	assert.NoError(t, err)
	assert.NotEmpty(t, newToken)
	assert.NotEqual(t, originalToken, newToken)

	// 验证新 token
	newClaims, err := jv.ParseToken(newToken)
	assert.NoError(t, err)
	assert.Equal(t, originalClaims.UserID, newClaims.UserID)
	assert.Equal(t, originalClaims.Username, newClaims.Username)
	assert.Equal(t, originalClaims.Data["role"], newClaims.Data["role"])

	// 验证新 token 的过期时间更晚
	assert.True(t, newClaims.ExpiresAt.After(originalClaims.ExpiresAt.Time))
}

func TestJwtValidator_AutoRefresh(t *testing.T) {
	setupTest()

	// 设置短的过期时间和续期阈值用于测试
	config.SetDefault("security.jwt.expire_time", 10)      // 10 秒过期
	config.SetDefault("security.jwt.refresh_threshold", 8) // 剩余 8 秒时续期
	config.SetDefault("security.jwt.auto_refresh", true)   // 确保自动续期开启

	jv := NewJwtValidator()
	gin.SetMode(gin.TestMode)

	// 创建一个即将过期的 token（剩余 5 秒）
	claims := &Claims{
		UserID:   "user123",
		Username: "john_doe",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(5 * time.Second)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString([]byte("test-secret-key"))

	// 创建测试请求和响应
	w := httptest.NewRecorder()

	// 创建实际的 gin engine 来处理响应
	router := gin.New()
	router.GET("/test", func(ctx *gin.Context) {
		// 直接调用 checkAndRefresh
		parsedClaims, _ := jv.ParseToken(tokenString)
		jv.checkAndRefresh(ctx, parsedClaims)
		ctx.String(200, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	router.ServeHTTP(w, req)

	// 验证响应头中是否包含新 token
	newToken := w.Header().Get("X-New-Token")
	assert.NotEmpty(t, newToken, "Auto-refresh should generate a new token")
	assert.NotEqual(t, tokenString, newToken)
}

func TestJwtValidator_TokenFromMultipleSources(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 测试从 header 获取
	t.Run("FromHeader", func(t *testing.T) {
		setupTest()
		config.SetDefault("security.jwt.token_lookup", "header:Authorization")
		jv := NewJwtValidator()
		token, _ := jv.GenerateToken("user123", "john_doe", nil)

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)
		c.Request.Header.Set("Authorization", "Bearer "+token)

		err := jv.Valid(c)
		assert.NoError(t, err)
	})

	// 测试从 query 获取
	t.Run("FromQuery", func(t *testing.T) {
		setupTest()
		config.SetDefault("security.jwt.token_lookup", "query:token")
		jv := NewJwtValidator()
		token, _ := jv.GenerateToken("user123", "john_doe", nil)

		// 使用 gin router 来正确处理 query 参数
		router := gin.New()
		var testErr error
		router.GET("/test", func(c *gin.Context) {
			testErr = jv.Valid(c)
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/test?token="+token, nil)
		router.ServeHTTP(w, req)

		assert.NoError(t, testErr)
	})

	// 测试从 cookie 获取
	t.Run("FromCookie", func(t *testing.T) {
		setupTest()
		config.SetDefault("security.jwt.token_lookup", "cookie:jwt_token")
		jv := NewJwtValidator()
		token, _ := jv.GenerateToken("user123", "john_doe", nil)

		// 使用 gin router 来正确处理 cookie
		router := gin.New()
		var testErr error
		router.GET("/test", func(c *gin.Context) {
			testErr = jv.Valid(c)
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.AddCookie(&http.Cookie{
			Name:  "jwt_token",
			Value: token,
		})
		router.ServeHTTP(w, req)

		assert.NoError(t, testErr)
	})
}

func TestJwtValidator_DifferentSigningMethods(t *testing.T) {
	setupTest()

	testCases := []struct {
		name   string
		method string
	}{
		{"HS256", "HS256"},
		{"HS384", "HS384"},
		{"HS512", "HS512"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config.SetDefault("security.jwt.signing_method", tc.method)
			jv := NewJwtValidator()

			token, err := jv.GenerateToken("user123", "john_doe", nil)
			assert.NoError(t, err)

			claims, err := jv.ParseToken(token)
			assert.NoError(t, err)
			assert.Equal(t, "user123", claims.UserID)
		})
	}
}

func TestGetHelperFunctions(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 创建测试 context
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// 设置 claims
	claims := &Claims{
		UserID:   "user123",
		Username: "john_doe",
	}
	c.Set(JwtContextKey, claims)

	// 测试 GetClaims
	gotClaims, exists := GetClaims(c)
	assert.True(t, exists)
	assert.Equal(t, claims, gotClaims)

	// 测试 GetUserID
	userID := GetUserID(c)
	assert.Equal(t, "user123", userID)

	// 测试 GetUsername
	username := GetUsername(c)
	assert.Equal(t, "john_doe", username)

	// 测试没有 claims 的情况
	w2 := httptest.NewRecorder()
	c2, _ := gin.CreateTestContext(w2)

	_, exists = GetClaims(c2)
	assert.False(t, exists)

	assert.Empty(t, GetUserID(c2))
	assert.Empty(t, GetUsername(c2))
}

// 辅助函数：创建过期的 token
func createExpiredToken(t *testing.T, jv *JwtValidator) string {
	claims := &Claims{
		UserID:   "user123",
		Username: "john_doe",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)), // 1 小时前过期
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
			NotBefore: jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		},
	}

	token := jwt.NewWithClaims(jv.signingMethod, claims)
	tokenString, err := token.SignedString([]byte(jv.config.PrivateKey))
	assert.NoError(t, err)

	return tokenString
}
