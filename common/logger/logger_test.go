package logger

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/trace"
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

// ============== Tracing Tests ==============

// TestTracingWithMiddleware tests context.Value extraction (backward compatible)
func TestTracingWithMiddleware(t *testing.T) {
	preferOtel := true
	Setup(func(cfg *Config) {
		cfg.Level = "info"
		cfg.Tracing = &TracingConfig{
			Enabled:        true,
			PreferOtelSpan: &preferOtel,
		}
	})

	// Simulate middleware adding trace_id via context.Value
	ctx := context.Background()
	ctx = WithTraceID(ctx, "middleware-trace-123")
	ctx = WithSpanID(ctx, "middleware-span-456")

	// Verify extraction
	if GetTraceID(ctx) != "middleware-trace-123" {
		t.Errorf("Expected middleware-trace-123, got=%s", GetTraceID(ctx))
	}

	TInfo(ctx, "Middleware trace test")
}

// TestTracingWithOtelSpan tests OpenTelemetry span extraction with auto-initialized TracerProvider
func TestTracingWithOtelSpan(t *testing.T) {
	// Simple setup - TracerProvider auto-initialized
	Setup(func(cfg *Config) {
		cfg.Level = "info"
		cfg.Tracing = &TracingConfig{
			Enabled: true, // Auto-initialization
		}
	})

	// Create otel span
	ctx := context.Background()
	ctx, span := StartSpan(ctx, "test-operation")
	defer span.End()

	// Verify span is valid
	if !span.SpanContext().IsValid() {
		t.Error("Expected valid span with auto-initialized TracerProvider")
	}

	traceID := GetTraceID(ctx)
	spanID := GetSpanID(ctx)

	if traceID == "" || spanID == "" {
		t.Error("Expected non-empty trace_id and span_id")
	}

	TInfo(ctx, "Otel span test")
}

// TestTracingPriority tests priority control when both sources exist
func TestTracingPriority(t *testing.T) {
	tp := trace.NewTracerProvider()
	otel.SetTracerProvider(tp)

	t.Run("PreferOtelSpan", func(t *testing.T) {
		preferOtel := true
		Setup(func(cfg *Config) {
			cfg.Level = "info"
			cfg.Tracing = &TracingConfig{
				Enabled:        true,
				PreferOtelSpan: &preferOtel,
				TracerProvider: tp,
			}
		})

		ctx := context.Background()
		ctx, span := StartSpan(ctx, "otel-op")
		defer span.End()

		// Also add via context.Value
		ctx = WithTraceID(ctx, "context-trace")

		// Should prefer otel span
		traceID := GetTraceID(ctx)
		spanCtx := span.SpanContext()
		if traceID != spanCtx.TraceID().String() {
			t.Error("Expected otel trace_id to be preferred")
		}

		TInfo(ctx, "Prefer otel test")
	})

	t.Run("PreferContextValue", func(t *testing.T) {
		preferContext := false
		Setup(func(cfg *Config) {
			cfg.Level = "info"
			cfg.Tracing = &TracingConfig{
				Enabled:        true,
				PreferOtelSpan: &preferContext,
				TracerProvider: tp,
			}
		})

		ctx := context.Background()
		ctx, span := StartSpan(ctx, "otel-op")
		defer span.End()

		ctx = WithTraceID(ctx, "context-trace-789")

		// Logger should prefer context.Value (check via log output)
		TInfo(ctx, "Prefer context test")
	})
}

// TestTracingDistributed tests distributed tracing with remote parent
func TestTracingDistributed(t *testing.T) {
	Setup(func(cfg *Config) {
		cfg.Level = "info"
		cfg.Tracing = &TracingConfig{Enabled: true}
	})

	// Simulate receiving parent trace from HTTP headers
	parentTraceID := "0123456789abcdef0123456789abcdef"
	parentSpanID := "0123456789abcdef"

	// Test InjectTraceContext
	ctx := context.Background()
	ctx, err := InjectTraceContext(ctx, parentTraceID, parentSpanID)
	if err != nil {
		t.Fatalf("InjectTraceContext failed: %v", err)
	}

	// Test StartSpanWithRemoteParent
	ctx, span, err := StartSpanWithRemoteParent(ctx, "child-op", parentTraceID, parentSpanID)
	if err != nil {
		t.Fatalf("StartSpanWithRemoteParent failed: %v", err)
	}
	defer span.End()

	if !span.SpanContext().IsValid() {
		t.Error("Expected valid span with remote parent")
	}

	TInfo(ctx, "Distributed tracing test")
}

// TestTracingDefaultProvider tests auto-initialization of TracerProvider
func TestTracingDefaultProvider(t *testing.T) {
	// Clear global state
	globalTracerProvider = nil

	// First logger - should auto-initialize
	logger1 := New(Config{
		Level: "info",
		Tracing: &TracingConfig{
			Enabled: true,
		},
	})

	if logger1.tracingConfig.TracerProvider == nil {
		t.Error("Expected TracerProvider to be auto-initialized")
	}

	// Second logger - should reuse global provider
	logger2 := New(Config{
		Level: "info",
		Tracing: &TracingConfig{
			Enabled: true,
		},
	})

	if logger1.tracingConfig.TracerProvider != logger2.tracingConfig.TracerProvider {
		t.Error("Expected both loggers to share global TracerProvider")
	}

	// Test actual usage
	ctx, span := StartSpan(context.Background(), "auto-init-test")
	defer span.End()

	if GetTraceID(ctx) == "" {
		t.Error("Expected trace_id with auto-initialized provider")
	}
}

// TestTracingDisabled tests behavior when tracing is disabled
func TestTracingDisabled(t *testing.T) {
	Setup(func(cfg *Config) {
		cfg.Level = "info"
		cfg.Tracing = &TracingConfig{
			Enabled: false,
		}
	})

	// Manual trace_id via context.Value should still work
	ctx := context.Background()
	ctx = WithTraceID(ctx, "manual-trace")

	if GetTraceID(ctx) != "manual-trace" {
		t.Error("GetTraceID should work when tracing disabled")
	}

	// But logger won't add trace fields automatically
	TInfo(ctx, "Tracing disabled test")
}
