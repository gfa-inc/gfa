package security

import (
	"errors"
	"github.com/gfa-inc/gfa/common/logger"
	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
	"strings"
)

var (
	ErrNotFoundHeaderApiKey = errors.New("not found api key in header")
	ApiKeyValidateHandler   func(c *gin.Context, apiKey string) error
)

type ApiKeyValidator struct {
	Lookup    string
	HeaderKey string
	QueryKey  string
}

func NewApiKeyValidator(headerKey string, lookup string) *ApiKeyValidator {
	apiKeyValidator := &ApiKeyValidator{
		HeaderKey: headerKey,
		Lookup:    lookup,
	}
	if apiKeyValidator.Lookup != "" {
		lookupItems := strings.Split(apiKeyValidator.Lookup, ",")
		for _, item := range lookupItems {
			trimmedItem := strings.TrimSpace(item)
			fields := strings.Split(trimmedItem, ":")

			if len(fields) != 2 {
				logger.Warnf("Invalid lookup item: %s", item)
				continue
			}

			trimmedFields := lo.Map(fields, func(item string, index int) string {
				return strings.TrimSpace(item)
			})

			switch trimmedFields[0] {
			case "header":
				apiKeyValidator.HeaderKey = trimmedFields[1]
			case "query":
				apiKeyValidator.QueryKey = trimmedFields[1]
			}
		}
	}
	return apiKeyValidator
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
