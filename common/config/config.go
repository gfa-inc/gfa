package config

import (
	"errors"
	"github.com/gfa-inc/gfa/common/logger"
	"github.com/spf13/viper"
)

type Config struct {
	ConfigName   string   `json:"configName"`
	ConfigType   string   `json:"configType"`
	AutomaticEnv bool     `json:"automaticEnv"`
	Paths        []string `json:"paths"`
}

type OptionFunc func(config *Config)

func WithPath(path string) OptionFunc {
	return func(config *Config) {
		config.Paths = append(config.Paths, path)
	}
}

func WithConfigName(name string) OptionFunc {
	return func(config *Config) {
		config.ConfigName = name
	}
}

func WithConfigType(configType string) OptionFunc {
	return func(config *Config) {
		config.ConfigType = configType
	}
}

func WithAutomaticEnv(flag bool) OptionFunc {
	return func(config *Config) {
		config.AutomaticEnv = flag
	}
}

func Setup(opts ...OptionFunc) {
	configLogger := logger.New(logger.Config{})

	config := &Config{
		ConfigName:   "application",
		ConfigType:   "yaml",
		AutomaticEnv: true,
	}
	for _, opt := range opts {
		opt(config)
	}

	viper.SetConfigName(config.ConfigName)
	viper.SetConfigType(config.ConfigType)
	if config.AutomaticEnv {
		viper.AutomaticEnv()
	}

	viper.AddConfigPath(".")
	for _, path := range config.Paths {
		viper.AddConfigPath(path)
	}

	err := viper.ReadInConfig()
	if err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if errors.As(err, &configFileNotFoundError) {
			configLogger.Panic(nil, "Config file not found")
			return
		}
		configLogger.Panicf(nil, "Fail to read config file, %s", err)
	}
}

func SetDefault(name string, value any) {
	viper.SetDefault(name, value)
}

func GetString(key string) string {
	return viper.GetString(key)
}

func GetInt(key string) int {
	return viper.GetInt(key)
}

func GetBool(key string) bool {
	return viper.GetBool(key)
}

func GetFloat64(key string) float64 {
	return viper.GetFloat64(key)
}

func GetStringSlice(key string) []string {
	return viper.GetStringSlice(key)
}

func GetStringMap(key string) map[string]interface{} {
	return viper.GetStringMap(key)
}

func GetStringMapString(key string) map[string]string {
	return viper.GetStringMapString(key)
}

func GetStringMapStringSlice(key string) map[string][]string {
	return viper.GetStringMapStringSlice(key)
}

func Get(key string) any {
	return viper.Get(key)
}

func UnmarshalKey(key string, rawVal any, opts ...viper.DecoderConfigOption) error {
	return viper.UnmarshalKey(key, rawVal, opts...)
}

func Unmarshal(rawVal any, opts ...viper.DecoderConfigOption) error {
	return viper.Unmarshal(rawVal, opts...)
}
