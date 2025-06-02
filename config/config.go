package config

import (
	"github.com/rs/zerolog"
	"github.com/spf13/viper"
	"os"
	"reflect"
	"strings"
)

const (
	EnvPrefix = "OMS"
)

type Config struct {
	Database struct {
		URI string `yaml:"uri" mapstructure:"uri"`
	} `yaml:"database" mapstructure:"database"`
	Server struct {
		Port int `mapstructure:"port"`
	} `yaml:"server" mapstructure:"server"`
}

func Get(logger zerolog.Logger) *Config {
	v := viper.New()
	v.SetEnvPrefix(EnvPrefix)
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AddConfigPath("./config/")
	v.SetConfigName("config")
	v.SetConfigType("yaml")

	err := v.ReadInConfig()
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to read config")
	}

	for _, key := range v.AllKeys() {
		val := v.Get(key)
		if val == nil {
			continue
		}

		if reflect.TypeOf(val).Kind() == reflect.String {
			v.Set(key, os.ExpandEnv(val.(string)))
		}
	}

	var cfg *Config
	err = v.Unmarshal(&cfg)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to unmarshal config")
	}

	return cfg
}
