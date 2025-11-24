package logger

import (
	"context"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/gookit/color"
	"github.com/samber/lo"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/trace"
	oteltrace "go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
)

var (
	globalLogger         *Logger
	coreMap              map[string]coreFactory
	globalTracerConfig   *TracingConfig
	globalTracerProvider *trace.TracerProvider
)

type coreFactory func(option Config) zapcore.Core

// Tracing keys
const (
	TraceIDKey = "trace_id"
	SpanIDKey  = "span_id"
)

// TracingConfig controls distributed tracing behavior
type TracingConfig struct {
	// Enabled enables tracing integration
	Enabled bool

	// PreferOtelSpan: if true, prefer extracting from otel span over context.Value
	// nil means default to true (use otel span first, fallback to context.Value)
	// Set to &false explicitly if you want to prefer context.Value
	PreferOtelSpan *bool

	// TracerProvider: custom OpenTelemetry TracerProvider
	// If nil, a default TracerProvider will be created automatically in New()
	TracerProvider oteltrace.TracerProvider

	// TracerName: tracer name for otel.Tracer()
	// Default: "default"
	TracerName string
}

type Config struct {
	ServiceName   string            // service name
	Level         string            // logger level
	CtxKeys       []string          // context key which will be logged from context
	CtxKeyMapping map[string]string // context key mapping
	Tracing       *TracingConfig    // tracing configuration (optional)
}

type Logger struct {
	inner         *zap.SugaredLogger
	level         zapcore.Level
	CtxKeys       []string
	CtxKeyMapping map[string]string
	tracingConfig *TracingConfig
}

func New(option Config) *Logger {
	level, err := zapcore.ParseLevel(option.Level)
	if err != nil {
		log.Panic(err)
	}

	l := &Logger{
		level:         level,
		CtxKeys:       option.CtxKeys,
		CtxKeyMapping: option.CtxKeyMapping,
		tracingConfig: initializeTracingConfig(option.Tracing),
	}
	core := zapcore.NewTee(lo.Map(lo.Entries(coreMap), func(entry lo.Entry[string, coreFactory], _ int) zapcore.Core {
		return entry.Value(option)
	})...)

	l.inner = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(2),
		zap.AddStacktrace(zap.NewAtomicLevelAt(zapcore.ErrorLevel))).Sugar()

	return l
}

// initializeTracingConfig initializes tracing configuration with defaults
// Tracing is enabled by default if not explicitly configured
func initializeTracingConfig(cfg *TracingConfig) *TracingConfig {
	// If no config provided, enable tracing by default
	if cfg == nil {
		defaultTrue := true
		cfg = &TracingConfig{
			Enabled:        true, // Default: enabled
			PreferOtelSpan: &defaultTrue,
		}
	}

	// Set default PreferOtelSpan if not specified
	if cfg.PreferOtelSpan == nil {
		defaultTrue := true
		cfg.PreferOtelSpan = &defaultTrue
	}

	// Initialize default TracerProvider if tracing is enabled and provider not set
	if cfg.Enabled && cfg.TracerProvider == nil {
		if globalTracerProvider == nil {
			globalTracerProvider = trace.NewTracerProvider(
				trace.WithSampler(trace.AlwaysSample()),
			)
			otel.SetTracerProvider(globalTracerProvider)
		}
		cfg.TracerProvider = globalTracerProvider
	}

	return cfg
}

func NewBasic(option Config) *Logger {
	level, err := zapcore.ParseLevel(option.Level)
	if err != nil {
		log.Panic(err)
	}

	l := &Logger{
		level: level,
	}

	core := zapcore.NewTee(coreMap["console"](option))

	l.inner = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(2),
		zap.AddStacktrace(zap.NewAtomicLevelAt(zapcore.ErrorLevel))).Sugar()

	return l
}

func (l *Logger) AddContextKey(key string) {
	l.CtxKeys = append(l.CtxKeys, key)
}

func (l *Logger) Clone(level zapcore.Level) Logger {
	cloned := *l
	cloned.level = level
	cloned.CtxKeys = l.CtxKeys
	cloned.CtxKeyMapping = l.CtxKeyMapping
	return cloned
}

func (l *Logger) getKeyName(key string) string {
	if l.CtxKeyMapping != nil {
		if v, ok := l.CtxKeyMapping[key]; ok {
			return v
		}
	}

	return key
}

func (l *Logger) WithContext(ctx context.Context) *zap.SugaredLogger {
	if ctx == nil {
		return l.inner
	}

	newLogger := l.inner

	// Handle tracing if enabled
	if l.tracingConfig != nil && l.tracingConfig.Enabled {
		traceID, spanID := l.extractTraceInfo(ctx)

		if traceID != "" {
			newLogger = newLogger.With(l.getKeyName(TraceIDKey), traceID)
		}
		if spanID != "" {
			newLogger = newLogger.With(l.getKeyName(SpanIDKey), spanID)
		}
	}

	// Add other context keys
	for _, v := range l.CtxKeys {
		// Skip trace_id/span_id if already handled by tracing config
		if l.tracingConfig != nil && l.tracingConfig.Enabled {
			if v == TraceIDKey || v == SpanIDKey {
				continue
			}
		}

		if val := ctx.Value(v); val != nil {
			newLogger = newLogger.With(l.getKeyName(v), val)
		}
	}

	return newLogger
}

// extractTraceInfo intelligently extracts trace_id and span_id from context
// Priority: trace_id from context.Value first, span_id from oteltrace first
func (l *Logger) extractTraceInfo(ctx context.Context) (traceID string, spanID string) {
	// trace_id: prioritize context.Value
	if v := ctx.Value(TraceIDKey); v != nil {
		if s, ok := v.(string); ok && s != "" {
			traceID = s
		}
	}
	// If not found in context.Value, try otel span
	if traceID == "" {
		span := oteltrace.SpanFromContext(ctx)
		if span.SpanContext().IsValid() {
			traceID = span.SpanContext().TraceID().String()
		}
	}

	// span_id: prioritize oteltrace
	span := oteltrace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		spanID = span.SpanContext().SpanID().String()
	}
	// If not found in otel span, try context.Value
	if spanID == "" {
		if v := ctx.Value(SpanIDKey); v != nil {
			if s, ok := v.(string); ok {
				spanID = s
			}
		}
	}

	return traceID, spanID
}

func (l *Logger) Printf(ctx context.Context, format string, args ...any) {
	l.WithContext(ctx).Infof(format, args...)
}

func (l *Logger) Info(ctx context.Context, args ...any) {
	if !l.level.Enabled(zapcore.InfoLevel) {
		return
	}
	l.WithContext(ctx).Infoln(args...)
}

func (l *Logger) Infof(ctx context.Context, format string, args ...any) {
	if !l.level.Enabled(zapcore.InfoLevel) {
		return
	}
	l.WithContext(ctx).Infof(format, args...)
}

func (l *Logger) Warn(ctx context.Context, args ...any) {
	if !l.level.Enabled(zapcore.WarnLevel) {
		return
	}
	l.WithContext(ctx).Warnln(args...)
}

func (l *Logger) Warnf(ctx context.Context, format string, args ...any) {
	if !l.level.Enabled(zapcore.WarnLevel) {
		return
	}
	l.WithContext(ctx).Warnf(format, args...)
}

func (l *Logger) Error(ctx context.Context, args ...any) {
	if !l.level.Enabled(zapcore.ErrorLevel) {
		return
	}
	l.WithContext(ctx).Errorln(args...)
}

func (l *Logger) Errorf(ctx context.Context, format string, args ...any) {
	if !l.level.Enabled(zapcore.ErrorLevel) {
		return
	}
	l.WithContext(ctx).Errorf(format, args...)
}

func (l *Logger) Debug(ctx context.Context, args ...any) {
	if !l.level.Enabled(zapcore.DebugLevel) {
		return
	}
	l.WithContext(ctx).Debugln(args...)
}

func (l *Logger) Debugf(ctx context.Context, format string, args ...any) {
	if !l.level.Enabled(zapcore.DebugLevel) {
		return
	}
	l.WithContext(ctx).Debugf(format, args...)
}

func (l *Logger) Fatal(ctx context.Context, args ...any) {
	if !l.level.Enabled(zapcore.FatalLevel) {
		return
	}
	l.WithContext(ctx).Fatal(args...)
}

func (l *Logger) Fatalf(ctx context.Context, format string, args ...any) {
	if !l.level.Enabled(zapcore.FatalLevel) {
		return
	}
	l.WithContext(ctx).Fatalf(format, args...)
}

func (l *Logger) Panic(ctx context.Context, args ...any) {
	if !l.level.Enabled(zapcore.PanicLevel) {
		return
	}
	l.WithContext(ctx).Panic(args...)
}

func (l *Logger) Panicf(ctx context.Context, format string, args ...any) {
	if !l.level.Enabled(zapcore.PanicLevel) {
		return
	}
	l.WithContext(ctx).Panicf(format, args...)
}

func (l *Logger) LevelEnabled(levelText string) bool {
	level, err := zapcore.ParseLevel(levelText)
	if err != nil {
		return false
	}
	return l.inner.Level().Enabled(level)
}

func RegisterCore(name string, coreFactory func(option Config) zapcore.Core) {
	coreMap[name] = coreFactory
}

func colorByLevel(level zapcore.Level) color.Color {
	switch level {
	case zapcore.DebugLevel:
		return color.FgBlue
	case zapcore.InfoLevel:
		return color.FgGreen
	case zapcore.WarnLevel:
		return color.FgYellow
	case zapcore.ErrorLevel, zapcore.DPanicLevel, zapcore.PanicLevel, zapcore.FatalLevel:
		return color.FgRed
	default:
		return color.FgWhite
	}
}

// kvConsoleEncoder wraps zapcore encoder to output fields in key=value format
type kvConsoleEncoder struct {
	zapcore.Encoder
	cfg    zapcore.EncoderConfig
	fields map[string]interface{}
}

func newKVConsoleEncoder(cfg zapcore.EncoderConfig) *kvConsoleEncoder {
	return &kvConsoleEncoder{
		Encoder: zapcore.NewConsoleEncoder(cfg),
		cfg:     cfg,
		fields:  make(map[string]interface{}),
	}
}

func (enc *kvConsoleEncoder) Clone() zapcore.Encoder {
	clone := &kvConsoleEncoder{
		Encoder: enc.Encoder.Clone(),
		cfg:     enc.cfg,
		fields:  make(map[string]interface{}, len(enc.fields)),
	}
	// Copy existing fields
	for k, v := range enc.fields {
		clone.fields[k] = v
	}
	return clone
}

// All Add* methods simply store the value in the map
func (enc *kvConsoleEncoder) AddString(k, v string)                 { enc.fields[k] = v }
func (enc *kvConsoleEncoder) AddBool(k string, v bool)              { enc.fields[k] = v }
func (enc *kvConsoleEncoder) AddInt(k string, v int)                { enc.fields[k] = v }
func (enc *kvConsoleEncoder) AddInt64(k string, v int64)            { enc.fields[k] = v }
func (enc *kvConsoleEncoder) AddInt32(k string, v int32)            { enc.fields[k] = v }
func (enc *kvConsoleEncoder) AddInt16(k string, v int16)            { enc.fields[k] = v }
func (enc *kvConsoleEncoder) AddInt8(k string, v int8)              { enc.fields[k] = v }
func (enc *kvConsoleEncoder) AddUint(k string, v uint)              { enc.fields[k] = v }
func (enc *kvConsoleEncoder) AddUint64(k string, v uint64)          { enc.fields[k] = v }
func (enc *kvConsoleEncoder) AddUint32(k string, v uint32)          { enc.fields[k] = v }
func (enc *kvConsoleEncoder) AddUint16(k string, v uint16)          { enc.fields[k] = v }
func (enc *kvConsoleEncoder) AddUint8(k string, v uint8)            { enc.fields[k] = v }
func (enc *kvConsoleEncoder) AddUintptr(k string, v uintptr)        { enc.fields[k] = v }
func (enc *kvConsoleEncoder) AddFloat64(k string, v float64)        { enc.fields[k] = v }
func (enc *kvConsoleEncoder) AddFloat32(k string, v float32)        { enc.fields[k] = v }
func (enc *kvConsoleEncoder) AddComplex128(k string, v complex128)  { enc.fields[k] = v }
func (enc *kvConsoleEncoder) AddComplex64(k string, v complex64)    { enc.fields[k] = v }
func (enc *kvConsoleEncoder) AddDuration(k string, v time.Duration) { enc.fields[k] = v }
func (enc *kvConsoleEncoder) AddTime(k string, v time.Time) {
	enc.fields[k] = v.Format(time.RFC3339Nano)
}
func (enc *kvConsoleEncoder) AddBinary(k string, v []byte)     { enc.fields[k] = fmt.Sprintf("%x", v) }
func (enc *kvConsoleEncoder) AddByteString(k string, v []byte) { enc.fields[k] = string(v) }
func (enc *kvConsoleEncoder) AddReflected(k string, v interface{}) error {
	enc.fields[k] = v
	return nil
}
func (enc *kvConsoleEncoder) AddArray(k string, v zapcore.ArrayMarshaler) error {
	enc.fields[k] = v
	return nil
}
func (enc *kvConsoleEncoder) AddObject(k string, v zapcore.ObjectMarshaler) error {
	enc.fields[k] = v
	return nil
}
func (enc *kvConsoleEncoder) OpenNamespace(string) {}

func (enc *kvConsoleEncoder) EncodeEntry(entry zapcore.Entry, fields []zapcore.Field) (*buffer.Buffer, error) {
	line := buffer.NewPool().Get()

	// Helper to encode time/level/caller
	encodeField := func(fn func(zapcore.PrimitiveArrayEncoder)) string {
		arr := &arrayEncoder{}
		fn(arr)
		if len(arr.elems) > 0 {
			return arr.elems[0]
		}
		return ""
	}

	// Encode time
	if enc.cfg.TimeKey != "" && enc.cfg.EncodeTime != nil {
		if s := encodeField(func(arr zapcore.PrimitiveArrayEncoder) {
			enc.cfg.EncodeTime(entry.Time, arr)
		}); s != "" {
			line.AppendString(s)
			line.AppendByte(' ')
		}
	}

	// Encode level
	if enc.cfg.LevelKey != "" && enc.cfg.EncodeLevel != nil {
		if s := encodeField(func(arr zapcore.PrimitiveArrayEncoder) {
			enc.cfg.EncodeLevel(entry.Level, arr)
		}); s != "" {
			line.AppendString(s)
			line.AppendByte(' ')
		}
	}

	// Encode caller
	if entry.Caller.Defined && enc.cfg.CallerKey != "" && enc.cfg.EncodeCaller != nil {
		if s := encodeField(func(arr zapcore.PrimitiveArrayEncoder) {
			enc.cfg.EncodeCaller(entry.Caller, arr)
		}); s != "" {
			line.AppendString(s)
			line.AppendByte(' ')
		}
	}

	// Add message
	if entry.Message != "" {
		line.AppendString(entry.Message)
	}

	// Collect fields
	for _, field := range fields {
		field.AddTo(enc)
	}

	// Output fields in key=value format
	if len(enc.fields) > 0 {
		// Get sorted keys for consistent output
		keys := make([]string, 0, len(enc.fields))
		for k := range enc.fields {
			keys = append(keys, k)
		}

		for _, k := range keys {
			line.AppendByte(' ')
			line.AppendString(fmt.Sprintf("%s=%v", k, enc.fields[k]))
		}

		// Clear fields for next use
		enc.fields = make(map[string]interface{})
	}

	// Add stack trace
	if entry.Stack != "" {
		line.AppendByte('\n')
		line.AppendString(entry.Stack)
	}

	line.AppendString(zapcore.DefaultLineEnding)
	return line, nil
}

// arrayEncoder is a simple helper to capture encoded values
type arrayEncoder struct{ elems []string }

func (a *arrayEncoder) AppendString(v string)          { a.elems = append(a.elems, v) }
func (a *arrayEncoder) AppendBool(v bool)              { a.elems = append(a.elems, fmt.Sprint(v)) }
func (a *arrayEncoder) AppendByteString(v []byte)      { a.elems = append(a.elems, string(v)) }
func (a *arrayEncoder) AppendComplex128(v complex128)  { a.elems = append(a.elems, fmt.Sprint(v)) }
func (a *arrayEncoder) AppendComplex64(v complex64)    { a.elems = append(a.elems, fmt.Sprint(v)) }
func (a *arrayEncoder) AppendFloat64(v float64)        { a.elems = append(a.elems, fmt.Sprint(v)) }
func (a *arrayEncoder) AppendFloat32(v float32)        { a.elems = append(a.elems, fmt.Sprint(v)) }
func (a *arrayEncoder) AppendInt(v int)                { a.elems = append(a.elems, fmt.Sprint(v)) }
func (a *arrayEncoder) AppendInt64(v int64)            { a.elems = append(a.elems, fmt.Sprint(v)) }
func (a *arrayEncoder) AppendInt32(v int32)            { a.elems = append(a.elems, fmt.Sprint(v)) }
func (a *arrayEncoder) AppendInt16(v int16)            { a.elems = append(a.elems, fmt.Sprint(v)) }
func (a *arrayEncoder) AppendInt8(v int8)              { a.elems = append(a.elems, fmt.Sprint(v)) }
func (a *arrayEncoder) AppendUint(v uint)              { a.elems = append(a.elems, fmt.Sprint(v)) }
func (a *arrayEncoder) AppendUint64(v uint64)          { a.elems = append(a.elems, fmt.Sprint(v)) }
func (a *arrayEncoder) AppendUint32(v uint32)          { a.elems = append(a.elems, fmt.Sprint(v)) }
func (a *arrayEncoder) AppendUint16(v uint16)          { a.elems = append(a.elems, fmt.Sprint(v)) }
func (a *arrayEncoder) AppendUint8(v uint8)            { a.elems = append(a.elems, fmt.Sprint(v)) }
func (a *arrayEncoder) AppendUintptr(v uintptr)        { a.elems = append(a.elems, fmt.Sprint(v)) }
func (a *arrayEncoder) AppendDuration(v time.Duration) { a.elems = append(a.elems, fmt.Sprint(v)) }
func (a *arrayEncoder) AppendTime(v time.Time)         { a.elems = append(a.elems, fmt.Sprint(v)) }

func consoleCoreFactory(option Config) zapcore.Core {
	cfg := zap.NewProductionEncoderConfig()
	cfg.EncodeTime = func(time time.Time, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString(color.New(color.OpFuzzy).Sprint(time.Format("2006-01-02T15:04:05.999")))
	}
	cfg.EncodeLevel = func(level zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString(color.New(colorByLevel(level), color.OpBold).Sprintf("%-5s", level.CapitalString()))
	}
	cfg.EncodeCaller = func(caller zapcore.EntryCaller, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString(color.New(color.OpFuzzy).Sprintf("%s:", caller.FullPath()))
	}
	cfg.ConsoleSeparator = " "

	// Use custom kvConsoleEncoder to output fields in key=value format
	consoleEncoder := newKVConsoleEncoder(cfg)
	consoleSync := zapcore.AddSync(os.Stdout)

	level, err := zapcore.ParseLevel(option.Level)
	if err != nil {
		log.Fatal(err)
	}
	return zapcore.NewCore(consoleEncoder, consoleSync, level)
}

type JsonCoreWriter struct {
	writeFunc func(b []byte) (int, error)
}

func NewJsonCoreWriter(writeFunc func(b []byte) (int, error)) *JsonCoreWriter {
	return &JsonCoreWriter{writeFunc: writeFunc}
}

func (w *JsonCoreWriter) Write(b []byte) (int, error) {
	return w.writeFunc(b)
}

func (w *JsonCoreWriter) Sync() error {
	return nil
}

func ToLevelPtr(level zapcore.Level) *zapcore.Level {
	return &level
}

type JsonCoreEncoderConfigFunc func(cfg *zapcore.EncoderConfig)

type NewJsonCoreOption struct {
	Writer       zapcore.WriteSyncer
	Level        *zapcore.Level
	LevelEnabler zapcore.LevelEnabler
	Opts         []JsonCoreEncoderConfigFunc
	Fields       func() []zapcore.Field
}

func NewJsonCore(option NewJsonCoreOption) func(option Config) zapcore.Core {
	cfg := zap.NewProductionEncoderConfig()
	for _, opt := range option.Opts {
		opt(&cfg)
	}
	jsonEncoder := zapcore.NewJSONEncoder(cfg)

	jsonSync := zapcore.AddSync(option.Writer)

	return func(c Config) zapcore.Core {
		if option.Level == nil {
			l, err := zapcore.ParseLevel(c.Level)
			if err != nil {
				log.Fatal(err)
			}
			option.Level = &l
		}

		if option.LevelEnabler == nil {
			levelEnabler := zap.LevelEnablerFunc(func(level zapcore.Level) bool {
				return level == *option.Level
			})
			option.LevelEnabler = levelEnabler
		}

		if option.Fields != nil {
			for _, field := range option.Fields() {
				field.AddTo(jsonEncoder)
			}
		}

		return zapcore.NewCore(jsonEncoder, jsonSync, option.LevelEnabler)
	}
}

func init() {
	coreMap = make(map[string]coreFactory)

	RegisterCore("console", consoleCoreFactory)
}

type OptionFunc func(option *Config)

func Setup(opts ...OptionFunc) {
	var option Config
	for _, opt := range opts {
		opt(&option)
	}
	globalLogger = New(option)
	globalTracerConfig = option.Tracing

	Debugf("Added logger core: %s", strings.Join(lo.Keys(coreMap), ", "))

	Infof("Global logger has been initialized with %d core", len(coreMap))
}

// Panic logs PanicLevel message
func Panic(args ...any) {
	globalLogger.Panic(nil, args...)
}

// Panicf logs PanicLevel message by format
func Panicf(format string, args ...any) {
	globalLogger.Panicf(nil, format, args...)
}

// Fatal logs FatalLevel message
func Fatal(args ...any) {
	globalLogger.Fatal(nil, args...)
}

// Fatalf logs FatalLevel message by format
func Fatalf(format string, args ...any) {
	globalLogger.Fatalf(nil, format, args...)
}

// Error logs ErrorLevel message
func Error(args ...any) {
	globalLogger.Error(nil, args...)
}

// Errorf logs ErrorLevel message by format
func Errorf(format string, args ...any) {
	globalLogger.Errorf(nil, format, args...)
}

// Warn logs WarnLevel message
func Warn(args ...any) {
	globalLogger.Warn(nil, args...)
}

// Warnf logs WarnLevel message
func Warnf(format string, args ...any) {
	globalLogger.Warnf(nil, format, args...)
}

// Info logs InfoLevel message
func Info(args ...any) {
	globalLogger.Info(nil, args...)
}

// Infof logs InfoLevel message by format
func Infof(format string, args ...any) {
	globalLogger.Infof(nil, format, args...)
}

// Debug logs DebugLevel message
func Debug(args ...any) {
	globalLogger.Debug(nil, args...)
}

// Debugf logs DebugLevel message by format
func Debugf(format string, args ...any) {
	globalLogger.Debugf(nil, format, args...)
}

func TPanic(c context.Context, args ...any) {
	globalLogger.Panic(c, args...)
}

// TPanicf logs PanicLevel message by format
func TPanicf(c context.Context, format string, args ...any) {
	globalLogger.Panicf(c, format, args...)
}

// TFatal logs FatalLevel message
func TFatal(c context.Context, args ...any) {
	globalLogger.Fatal(c, args...)
}

// TFatalf logs FatalLevel message by format
func TFatalf(c context.Context, format string, args ...any) {
	globalLogger.Fatalf(c, format, args...)
}

// TError logs ErrorLevel message
func TError(c context.Context, args ...any) {
	globalLogger.Error(c, args...)
}

// TErrorf logs ErrorLevel message by format
func TErrorf(c context.Context, format string, args ...any) {
	globalLogger.Errorf(c, format, args...)
}

// TWarn logs WarnLevel message
func TWarn(c context.Context, args ...any) {
	globalLogger.Warn(c, args...)
}

// TWarnf logs WarnLevel message
func TWarnf(c context.Context, format string, args ...any) {
	globalLogger.Warnf(c, format, args...)
}

// TInfo logs InfoLevel message
func TInfo(c context.Context, args ...any) {
	globalLogger.Info(c, args...)
}

// TInfof logs InfoLevel message by format
func TInfof(c context.Context, format string, args ...any) {
	globalLogger.Infof(c, format, args...)
}

// TDebug logs DebugLevel message
func TDebug(c context.Context, args ...any) {
	globalLogger.Debug(c, args...)
}

// TDebugf logs DebugLevel message by format
func TDebugf(c context.Context, format string, args ...any) {
	globalLogger.Debugf(c, format, args...)
}

func GetGlobalLogger() *Logger {
	return globalLogger
}

func IsDebugLevelEnabled() bool {
	return globalLogger.inner.Level().Enabled(zap.DebugLevel)
}

func AddContextKey(key string) {
	if globalLogger != nil {
		globalLogger.AddContextKey(key)
	}
}

// ===== OpenTelemetry Tracing Helper Functions =====

// GetTracer returns the configured OpenTelemetry tracer
func GetTracer() oteltrace.Tracer {
	if globalTracerConfig == nil {
		return otel.GetTracerProvider().Tracer("default")
	}

	tracerName := globalTracerConfig.TracerName
	if tracerName == "" {
		tracerName = "default"
	}

	if globalTracerConfig.TracerProvider != nil {
		return globalTracerConfig.TracerProvider.Tracer(tracerName)
	}

	return otel.GetTracerProvider().Tracer(tracerName)
}

// StartSpan starts a new OpenTelemetry span
// Convenience wrapper around tracer.Start()
func StartSpan(ctx context.Context, spanName string, opts ...oteltrace.SpanStartOption) (context.Context, oteltrace.Span) {
	if ctx == nil {
		ctx = context.Background()
	}
	return GetTracer().Start(ctx, spanName, opts...)
}

// GetTraceID extracts trace_id from context (otel span or context.Value)
func GetTraceID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}

	// Try otel span first (standard way)
	span := oteltrace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		return span.SpanContext().TraceID().String()
	}

	// Fallback to context.Value (for middleware compatibility)
	if v := ctx.Value(TraceIDKey); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}

	return ""
}

// GetSpanID extracts span_id from context (otel span or context.Value)
func GetSpanID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}

	// Try otel span first (standard way)
	span := oteltrace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		return span.SpanContext().SpanID().String()
	}

	// Fallback to context.Value (for middleware compatibility)
	if v := ctx.Value(SpanIDKey); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}

	return ""
}

// WithTraceID adds trace_id to context using context.Value
// For middleware that extracts trace_id from HTTP headers
func WithTraceID(ctx context.Context, traceID string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, TraceIDKey, traceID)
}

// WithSpanID adds span_id to context using context.Value
// For middleware that extracts span_id from HTTP headers
func WithSpanID(ctx context.Context, spanID string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, SpanIDKey, spanID)
}

// InjectTraceContext creates a context with manually set trace_id/span_id as otel span context
// Useful for distributed tracing: receiving trace info from HTTP headers
func InjectTraceContext(ctx context.Context, traceIDHex, spanIDHex string) (context.Context, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	// Parse trace ID (32 hex chars)
	traceIDBytes, err := hex.DecodeString(traceIDHex)
	if err != nil || len(traceIDBytes) != 16 {
		return ctx, fmt.Errorf("invalid trace_id: must be 32 hex characters")
	}
	var traceID oteltrace.TraceID
	copy(traceID[:], traceIDBytes)

	// Parse span ID (16 hex chars)
	spanIDBytes, err := hex.DecodeString(spanIDHex)
	if err != nil || len(spanIDBytes) != 8 {
		return ctx, fmt.Errorf("invalid span_id: must be 16 hex characters")
	}
	var spanID oteltrace.SpanID
	copy(spanID[:], spanIDBytes)

	// Create span context
	spanContext := oteltrace.NewSpanContext(oteltrace.SpanContextConfig{
		TraceID:    traceID,
		SpanID:     spanID,
		TraceFlags: oteltrace.FlagsSampled,
	})

	// Inject into context
	return oteltrace.ContextWithSpanContext(ctx, spanContext), nil
}

// StartSpanWithRemoteParent starts a new span with a remote parent
// Extracts parent trace info from HTTP headers and creates a child span
func StartSpanWithRemoteParent(ctx context.Context, spanName string, parentTraceID, parentSpanID string, opts ...oteltrace.SpanStartOption) (context.Context, oteltrace.Span, error) {
	// Inject parent trace context
	ctx, err := InjectTraceContext(ctx, parentTraceID, parentSpanID)
	if err != nil {
		return ctx, nil, err
	}

	// Start child span
	ctx, span := StartSpan(ctx, spanName, opts...)
	return ctx, span, nil
}

// ExtractTraceInfo extracts trace_id and span_id as hex strings
func ExtractTraceInfo(ctx context.Context) (traceID string, spanID string) {
	return GetTraceID(ctx), GetSpanID(ctx)
}
