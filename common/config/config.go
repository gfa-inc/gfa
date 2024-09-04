package config

import (
	"fmt"
	"github.com/duke-git/lancet/v2/fileutil"
	"github.com/duke-git/lancet/v2/slice"
	"github.com/gfa-inc/gfa/common/logger"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
	"path/filepath"
)

var (
	kf *koanf.Koanf
)

type Config struct {
	ConfigName   string   `json:"configName"`
	ConfigType   []string `json:"configType"`
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
		if slice.IndexOf(config.ConfigType, configType) != -1 {
			config.ConfigType = append(config.ConfigType, configType)
		}
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
		ConfigType:   []string{"yaml", "yml"},
		AutomaticEnv: true,
	}
	for _, opt := range opts {
		opt(config)
	}

	supportedParsers := map[string]koanf.Parser{
		"yaml": yaml.Parser(),
		"yml":  yaml.Parser(),
	}

	kf = koanf.New(".")
	// load from env
	if config.AutomaticEnv {
		err := kf.Load(env.Provider("", ".", func(s string) string {
			return s
		}), nil)
		if err != nil {
			configLogger.Panicf(nil, "Fail to load environment variables, %s", err)
		}
	}
	// load from file
	// add current path
	if slice.IndexOf(config.Paths, ".") == -1 {
		config.Paths = append(config.Paths, ".")
	}
	for _, p := range config.Paths {
		for _, ct := range config.ConfigType {
			parser, ok := supportedParsers[ct]
			if !ok {
				configLogger.Panicf(nil, "Unsupported config type: %s", ct)
			}

			configFilePath, err := filepath.Abs(fmt.Sprintf("%s/%s.%s", p, config.ConfigName, ct))
			if err != nil {
				configLogger.Panicf(nil, "Fail to get config file path, %s", err)
			}

			if !fileutil.IsExist(configFilePath) {
				configLogger.Debugf(nil, "Config file %s not exist", configFilePath)
				continue
			}

			err = kf.Load(file.Provider(configFilePath), parser)
			if err != nil {
				configLogger.Panicf(nil, "Fail to load config file %s, %s", configFilePath, err)
				return
			}
		}

	}
}

func SetDefault(name string, value any) {
	err := kf.Load(confmap.Provider(map[string]interface{}{name: value}, "."), nil)
	if err != nil {
		logger.Panic(err)
		return
	}
}

func GetString(key string) string {
	return kf.String(key)
}

func GetInt(key string) int {
	return kf.Int(key)
}

func GetBool(key string) bool {
	return kf.Bool(key)
}

func GetFloat64(key string) float64 {
	return kf.Float64(key)
}

func GetStringSlice(key string) []string {
	return kf.Strings(key)
}

func GetStringMapString(key string) map[string]string {
	return kf.StringMap(key)
}

func GetStringMapStringSlice(key string) map[string][]string {
	return kf.StringsMap(key)
}

func Get(key string) any {
	return kf.Get(key)
}

type UnmarshalKeyOpt = func(conf *koanf.UnmarshalConf)

func UnmarshalKey(key string, rawVal any, opts ...UnmarshalKeyOpt) error {
	var conf koanf.UnmarshalConf
	for _, opt := range opts {
		opt(&conf)
	}
	return kf.UnmarshalWithConf(key, rawVal, conf)
}

func Raw() *koanf.Koanf {
	return kf
}
