package logger

import (
	"context"
	"github.com/gookit/color"
	"github.com/samber/lo"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"log"
	"os"
	"strings"
	"time"
)

var (
	globalLogger *Logger
	coreMap      map[string]coreFactory
)

type coreFactory func(option Config) zapcore.Core

type Config struct {
	ServiceName   string            // service name
	Level         string            // logger level
	CtxKeys       []string          // context key which will be logged from context
	CtxKeyMapping map[string]string // context key mapping
}

type Logger struct {
	inner         *zap.SugaredLogger
	level         zapcore.Level
	CtxKeys       []string
	CtxKeyMapping map[string]string
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
	}
	core := zapcore.NewTee(lo.Map(lo.Entries(coreMap), func(entry lo.Entry[string, coreFactory], _ int) zapcore.Core {
		return entry.Value(option)
	})...)

	l.inner = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(2),
		zap.AddStacktrace(zap.NewAtomicLevelAt(zapcore.ErrorLevel))).Sugar()

	return l
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
	for _, v := range l.CtxKeys {
		if val := ctx.Value(v); val != nil {
			newLogger = newLogger.With(l.getKeyName(v), val)
		}
	}

	return newLogger
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

func consoleCoreFactory(option Config) zapcore.Core {
	cfg := zap.NewProductionEncoderConfig()
	cfg.EncodeTime = func(time time.Time, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString(color.New(color.OpFuzzy).Sprint(time.Format("2006-01-02T15:04:05.000")))
	}
	cfg.EncodeLevel = func(level zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString(color.New(colorByLevel(level), color.OpBold).Sprintf("%-5s", level.CapitalString()))
	}
	cfg.EncodeCaller = func(caller zapcore.EntryCaller, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString(color.New(color.OpFuzzy).Sprintf("%s:", caller.FullPath()))
	}
	cfg.ConsoleSeparator = " "

	consoleEncoder := zapcore.NewConsoleEncoder(cfg)
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
