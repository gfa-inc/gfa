package security

import (
	"errors"
	"github.com/gfa-inc/gfa/common/logger"
	"github.com/gin-gonic/gin"
)

var (
	ErrNotFoundHeaderApiKey = errors.New("not found api key in header")
	ApiKeyValidateHandler   func(c *gin.Context, apiKey string) error
)

type ApiKeyValidator struct {
	HeaderKey string
}

func NewApiKeyValidator(headerKey string) *ApiKeyValidator {
	return &ApiKeyValidator{
		HeaderKey: headerKey,
	}
}

func (akv *ApiKeyValidator) Valid(c *gin.Context) error {
	apiKey := c.GetHeader(akv.HeaderKey)
	if apiKey == "" {
		return ErrNotFoundHeaderApiKey
	}

	if ApiKeyValidateHandler == nil {
		logger.Panic("Unimplemented ApiKeyValidateHandler")
	}

	return ApiKeyValidateHandler(c, apiKey)
}
