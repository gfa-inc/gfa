package router

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestMatcher_Match(t *testing.T) {
	matcher := New()
	matcher.AddRoutes([]string{"/api/v1/*"})

	assert.True(t, matcher.Match("/api/v1/sys_user"))
	assert.True(t, matcher.Match("/api/v1/sys_user/detail"))
}

func BenchmarkMatcher_Match(b *testing.B) {
	matcher := New()
	matcher.AddRoutes([]string{"/api/v1/*"})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		matcher.Match("/api/v1/sys_user")
	}
}
