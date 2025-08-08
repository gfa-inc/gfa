package router

import (
	"testing"

	"github.com/gfa-inc/gfa/common/config"
	"github.com/gfa-inc/gfa/common/logger"
)

func TestRequestMatcher_Match(t *testing.T) {
	config.Setup(config.WithPath("../../"))
	logger.Setup()
	matcher := NewRequestMatcher()
	matcher.AddRoutes([][]any{
		{
			"/api/v1/sys_user",
			"GET",
		},
	})
	matcher.Match("/api/v1/sys_user", "GET")
}
