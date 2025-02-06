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
	CookieKey string
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
			case "cookie":
				apiKeyValidator.CookieKey = trimmedFields[1]
			}
		}
	}
	return apiKeyValidator
}

func (akv *ApiKeyValidator) Valid(c *gin.Context) error {
	apiKey := akv.GetApiKey(c)

	if apiKey == "" {
		return ErrNotFoundHeaderApiKey
	}

	if ApiKeyValidateHandler == nil {
		logger.Panic("Unimplemented ApiKeyValidateHandler")
	}

	return ApiKeyValidateHandler(c, apiKey)
}

func (akv *ApiKeyValidator) GetApiKey(c *gin.Context) string {
	headerApiKey := c.GetHeader(akv.HeaderKey)
	if headerApiKey != "" {
		return headerApiKey
	}

	queryApiKey := c.Query(akv.QueryKey)
	if queryApiKey != "" {
		return queryApiKey
	}

	cookieApiKey, err := c.Cookie(akv.CookieKey)
	if err == nil && cookieApiKey != "" {
		return cookieApiKey
	}

	return ""
}
