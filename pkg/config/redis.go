package config

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/livekit/protocol/logger"
)

type RedisConfig struct {
	Address      string `mapstructure:"address" yaml:"address,omitempty"`
	Username     string `mapstructure:"username" yaml:"username,omitempty"`
	Password     string `mapstructure:"password" yaml:"password,omitempty"`
	DB           int    `mapstructure:"db" yaml:"db,omitempty"`
	DialTimeout  int    `mapstructure:"dial_timeout" yaml:"dial_timeout,omitempty"`
	ReadTimeout  int    `mapstructure:"read_timeout" yaml:"read_timeout,omitempty"`
	WriteTimeout int    `mapstructure:"write_timeout" yaml:"write_timeout,omitempty"`

	MasterName          string   `mapstructure:"sentinel_master_name" yaml:"sentinel_master_name,omitempty"`
	SentinelUsername    string   `mapstructure:"sentinel_username" yaml:"sentinel_username,omitempty"`
	SentinelPassword    string   `mapstructure:"sentinel_password" yaml:"sentinel_password,omitempty"`
	SentinelAddresses   []string `mapstructure:"sentinel_addresses" yaml:"sentinel_addresses,omitempty"`
	ClusterAddresses    []string `mapstructure:"cluster_addresses" yaml:"cluster_addresses,omitempty"`
	ClusterMaxRedirects *int     `mapstructure:"cluster_max_redirects" yaml:"cluster_max_redirects,omitempty"`

	ChannelName string `mapstructure:"channel_name" yaml:"channel_name,omitempty"`
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
