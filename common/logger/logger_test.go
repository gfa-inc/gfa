package logger

import "testing"

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
