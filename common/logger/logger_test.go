package logger

import (
	"context"
	"testing"
)

func TestSetup(t *testing.T) {
	Setup(func(option *Config) {
		option.Level = "debug"
	})
	Debug("debug")
	Warn("warn")
	Info("info")
	Error("error")
	defer func() {
		Fatal("fatal")
	}()
	Panic("panic")
}

// TestKVFormatWithContext tests that console output uses key=value format
func TestKVFormatWithContext(t *testing.T) {
	Setup(func(option *Config) {
		option.ServiceName = "test-service"
		option.Level = "info"
		option.CtxKeys = []string{"request_id", "user_id"}
		option.CtxKeyMapping = map[string]string{
			"request_id": "req_id",
		}
	})

	// Create context with values
	ctx := context.Background()
	ctx = context.WithValue(ctx, "request_id", "123456")
	ctx = context.WithValue(ctx, "user_id", "user789")

	// These should output in key=value format: req_id=123456 user_id=user789
	TInfo(ctx, "Test message with context")
	TInfof(ctx, "Formatted message: %s", "test")
	TWarn(ctx, "Warning message")

	// Without context - no key-value pairs
	Info("Message without context")
}

// TestAddContextKey tests adding context keys dynamically
func TestAddContextKey(t *testing.T) {
	Setup(func(option *Config) {
		option.Level = "info"
	})

	// Add context key dynamically
	AddContextKey("session_id")
	AddContextKey("trace_id")

	ctx := context.Background()
	ctx = context.WithValue(ctx, "session_id", "sess-abc")
	ctx = context.WithValue(ctx, "trace_id", "trace-xyz")

	TInfo(ctx, "Testing dynamic context keys")
}
