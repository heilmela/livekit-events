package config

import (
	"errors"
	"fmt"

	"github.com/go-playground/validator"
	"github.com/spf13/viper"
)

type Config struct {
	LogLevel      string         `mapstructure:"log_level" yaml:"log_level,omitempty"`
	RedisConfig   *RedisConfig   `mapstructure:"redis" yaml:"redis,omitempty"`
	ServerConfig  *ServerConfig  `mapstructure:"server" yaml:"server,omitempty"`
	LivekitConfig *LivekitConfig `mapstructure:"livekit" yaml:"livekit" validate:"required"`
}

type ServerConfig struct {
	Port            uint32   `mapstructure:"port" yaml:"port,omitempty"`
	BindAddresse    string   `mapstructure:"bind_address" yaml:"bind_address,omitempty"`
	TrustedUpstream []string `mapstructure:"trusted_upstreams" yaml:"trusted_upstreams,omitempty"`
}

type LivekitConfig struct {
	ApiKey    string `mapstructure:"api_key" yaml:"api_key,omitempty" validate:"required"`
	ApiSecret string `mapstructure:"api_secret" yaml:"api_secret,omitempty" validate:"required"`
}

func NewConfig(path string) (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	if path != "" {
		viper.AddConfigPath(".")
	}
	viper.AutomaticEnv()

	viper.SetDefault("server.port", "3000")
	viper.SetDefault("server.bind_addresse", "0.0.0.0")
	viper.SetDefault("log_level", "debug")

	if err := viper.ReadInConfig(); err != nil {
		var cfgErr viper.ConfigFileNotFoundError
		if errors.As(err, &cfgErr) {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("unable to decode into struct: %w", err)
	}

	validate := validator.New()
	if err := validate.Struct(config); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &config, nil
}
