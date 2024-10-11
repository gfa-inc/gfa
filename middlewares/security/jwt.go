package security

import (
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"strings"
)

var (
	ErrJwtInvalidTokenLookup  = errors.New("invalid jwt token lookup")
	ErrJwtNotFoundQueryToken  = errors.New("not found jwt token in query")
	ErrJwtNotFoundHeaderToken = errors.New("not found jwt token in header")
)

type JwtValidatorConfig struct {
	TokenLookup     string // "header:Authorization" or "query:token"
	TokenHeaderName string
	PrivateKey      string
}

type JwtValidator struct {
	JwtValidatorConfig

	tokenLookupMap [][2]string
}

func NewJwtValidator() *JwtValidator {
	return &JwtValidator{}
}

func (j *JwtValidator) Valid(c *gin.Context) error {
	token, err := j.ParseToken(c)
	if err != nil {
		return err
	}

	_, err = jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		return []byte(j.PrivateKey), nil
	}, jwt.WithoutClaimsValidation())
	if err != nil {
		return err
	}

	return nil
}

func (j *JwtValidator) ParseTokenLookup() error {
	fields := strings.Split(j.TokenLookup, ",")
	j.tokenLookupMap = make([][2]string, len(fields))
	for _, field := range fields {
		parts := strings.Split(strings.TrimSpace(field), ":")
		if len(parts) != 2 {
			return ErrJwtInvalidTokenLookup
		}

		j.tokenLookupMap = append(j.tokenLookupMap, [2]string{parts[0], parts[1]})
	}
	return nil
}

func (j *JwtValidator) ParseToken(c *gin.Context) (token string, err error) {
	for _, parts := range j.tokenLookupMap {
		lookup, key := parts[0], parts[1]

		switch lookup {
		case "query":
			token, err = tokenFromQuery(c, key)
		case "header":
			fallthrough
		default:
			token, err = tokenFromHeader(c, key)
		}

		if err == nil {
			return
		}
	}
	return
}

func tokenFromQuery(c *gin.Context, key string) (string, error) {
	token := c.Query(key)
	if token == "" {
		return "", ErrJwtNotFoundQueryToken
	}
	return token, nil
}

func tokenFromHeader(c *gin.Context, key string) (string, error) {
	token := c.Request.Header.Get(key)
	if token == "" {
		return "", ErrJwtNotFoundHeaderToken
	}

	token = strings.Replace(token, "Bearer ", "", 1)
	return token, nil
}
