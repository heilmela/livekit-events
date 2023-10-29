package config

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/go-playground/validator"
	"github.com/go-redis/redis/v8"
	"github.com/kelseyhightower/envconfig"
	"github.com/livekit/protocol/logger"
	"gopkg.in/yaml.v2"
)

type Config struct {
	LogLevel string         `envconfig:"log_level" default:"debug" yaml:"log_level,omitempty"`
	Redis    *RedisConfig   `yaml:"redis,omitempty"`
	Server   *ServerConfig  `yaml:"server,omitempty"`
	Livekit  *LivekitConfig `yaml:"livekit" validate:"required"`
}

type ServerConfig struct {
	Port            uint32   `envconfig:"port" default:"3000" yaml:"port,omitempty"`
	BindAddress     string   `envconfig:"bind_address" default:"0.0.0.0" yaml:"bind_address,omitempty"`
	TrustedUpstream []string `envconfig:"trusted_upstreams" yaml:"trusted_upstreams,omitempty"`
}

type LivekitConfig struct {
	ApiKey    string `envconfig:"api_key" yaml:"api_key,omitempty" validate:"required"`
	ApiSecret string `envconfig:"api_secret" yaml:"api_secret,omitempty" validate:"required"`
}

type RedisConfig struct {
	Address             string   `envconfig:"address" yaml:"address,omitempty"`
	Username            string   `envconfig:"username" yaml:"username,omitempty"`
	Password            string   `envconfig:"password" yaml:"password,omitempty"`
	DB                  int      `envconfig:"db" yaml:"db,omitempty"`
	DialTimeout         int      `envconfig:"dial_timeout" yaml:"dial_timeout,omitempty"`
	ReadTimeout         int      `envconfig:"read_timeout" yaml:"read_timeout,omitempty"`
	WriteTimeout        int      `envconfig:"write_timeout" yaml:"write_timeout,omitempty"`
	MasterName          string   `envconfig:"sentinel_master_name" yaml:"sentinel_master_name,omitempty"`
	SentinelUsername    string   `envconfig:"sentinel_username" yaml:"sentinel_username,omitempty"`
	SentinelPassword    string   `envconfig:"sentinel_password" yaml:"sentinel_password,omitempty"`
	SentinelAddresses   []string `envconfig:"sentinel_addresses" yaml:"sentinel_addresses,omitempty"`
	ClusterAddresses    []string `envconfig:"cluster_addresses" yaml:"cluster_addresses,omitempty"`
	ClusterMaxRedirects *int     `envconfig:"cluster_max_redirects" yaml:"cluster_max_redirects,omitempty"`
	ChannelName         string   `envconfig:"channel_name" yaml:"channel_name" default:"livekit"`
}

func NewConfig(path string) (*Config, error) {
	var config Config

	if path != "" {
		file, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
		err = yaml.Unmarshal(file, &config)
		if err != nil {
			return nil, fmt.Errorf("unable to decode into struct: %w", err)
		}
	}

	if err := envconfig.Process("", &config); err != nil {
		return nil, fmt.Errorf("error processing environment variables: %w", err)
	}

	validate := validator.New()
	if err := validate.Struct(config); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	if config.Redis != nil && len(config.Redis.ChannelName) == 0 {
		return nil, fmt.Errorf("channel name is required when using redis")
	}

	return &config, nil
}

func NewRedisClient(conf *RedisConfig) (redis.UniversalClient, error) {
	var connectOptions *redis.UniversalOptions
	var client redis.UniversalClient

	if len(conf.SentinelAddresses) > 0 {
		logger.Infow("connecting to redis at %s", conf.SentinelAddresses)
		connectOptions = &redis.UniversalOptions{
			Addrs:            conf.SentinelAddresses,
			SentinelUsername: conf.SentinelUsername,
			SentinelPassword: conf.SentinelPassword,
			MasterName:       conf.MasterName,
			Username:         conf.Username,
			Password:         conf.Password,
			DB:               conf.DB,
			DialTimeout:      time.Duration(conf.DialTimeout) * time.Millisecond,
			ReadTimeout:      time.Duration(conf.ReadTimeout) * time.Millisecond,
			WriteTimeout:     time.Duration(conf.WriteTimeout) * time.Millisecond,
		}
	} else if len(conf.ClusterAddresses) > 0 {
		logger.Infow("connecting to redis at %s", conf.ClusterAddresses)
		redirects := 2
		if conf.ClusterMaxRedirects != nil {
			redirects = *conf.ClusterMaxRedirects
		}
		connectOptions = &redis.UniversalOptions{
			Addrs:        conf.ClusterAddresses,
			Username:     conf.Username,
			Password:     conf.Password,
			DB:           conf.DB,
			MaxRedirects: redirects,
		}
	} else {
		logger.Infow("connecting to redis at %s", conf.Address)
		connectOptions = &redis.UniversalOptions{
			Addrs:    []string{conf.Address},
			Username: conf.Username,
			Password: conf.Password,
			DB:       conf.DB,
		}
	}
	client = redis.NewUniversalClient(connectOptions)

	if err := client.Ping(context.Background()).Err(); err != nil {
		err = fmt.Errorf("could not connect to redis %w", err)
		return nil, err
	}

	return client, nil
}
