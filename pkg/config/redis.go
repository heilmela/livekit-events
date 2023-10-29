package config

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/livekit/protocol/logger"
	"gopkg.in/yaml.v2"
)

type RedisConfig struct {
	Address      string `yaml:"address,omitempty"`
	Username     string `yaml:"username,omitempty"`
	Password     string `yaml:"password,omitempty"`
	DB           int    `yaml:"db,omitempty"`
	DialTimeout  int    `yaml:"dial_timeout,omitempty"`
	ReadTimeout  int    `yaml:"read_timeout,omitempty"`
	WriteTimeout int    `yaml:"write_timeout,omitempty"`

	MasterName          string   `yaml:"sentinel_master_name,omitempty"`
	SentinelUsername    string   `yaml:"sentinel_username,omitempty"`
	SentinelPassword    string   `yaml:"sentinel_password,omitempty"`
	SentinelAddresses   []string `yaml:"sentinel_addresses,omitempty"`
	ClusterAddresses    []string `yaml:"cluster_addresses,omitempty"`
	ClusterMaxRedirects *int     `yaml:"cluster_max_redirects,omitempty"`

	ChannelName string `yaml:"channel_name,omitempty"`
}

func NewRedisConfig(path string) (*RedisConfig, error) {

	var conf RedisConfig

	if value, ok := os.LookupEnv("ADDRESS"); ok {
		conf.Address = value
	}

	if value, ok := os.LookupEnv("USERNAME"); ok {
		conf.Username = value
	}

	if value, ok := os.LookupEnv("PASSWORD"); ok {
		conf.Password = value
	}

	if value, ok := os.LookupEnv("DB"); ok {
		if db, err := strconv.Atoi(value); err == nil {
			conf.DB = db
		}
	}

	if value, ok := os.LookupEnv("MASTER_NAME"); ok {
		conf.MasterName = value
	}

	if value, ok := os.LookupEnv("SENTINEL_USERNAME"); ok {
		conf.SentinelUsername = value
	}

	if value, ok := os.LookupEnv("SENTINEL_PASSWORD"); ok {
		conf.SentinelPassword = value
	}

	if value, ok := os.LookupEnv("SENTINEL_ADDRESSES"); ok {
		conf.SentinelAddresses = strings.Split(value, ",")
	}

	if value, ok := os.LookupEnv("CLUSTER_ADDRESSES"); ok {
		conf.ClusterAddresses = strings.Split(value, ",")
	}

	if value, ok := os.LookupEnv("DIAL_TIMEOUT"); ok {
		if timeout, err := strconv.Atoi(value); err == nil {
			conf.DialTimeout = timeout
		}
	}

	if value, ok := os.LookupEnv("READ_TIMEOUT"); ok {
		if timeout, err := strconv.Atoi(value); err == nil {
			conf.ReadTimeout = timeout
		}
	}

	if value, ok := os.LookupEnv("WRITE_TIMEOUT"); ok {
		if timeout, err := strconv.Atoi(value); err == nil {
			conf.WriteTimeout = timeout
		}
	}

	if value, ok := os.LookupEnv("CHANNEL_NAME"); ok {
		conf.ChannelName = value
	} else {
		conf.ChannelName = "livekit-events"
	}

	if conf.DialTimeout == 0 {
		conf.DialTimeout = 5000
	}

	if conf.ReadTimeout == 0 {
		conf.ReadTimeout = 800
	}

	if conf.WriteTimeout == 0 {
		conf.WriteTimeout = 350
	}

	if path != "" {
		bytes, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		confString := string(bytes)
		decoder := yaml.NewDecoder(strings.NewReader(confString))

		if err := decoder.Decode(&conf); err != nil {
			return nil, fmt.Errorf("could not  redis config: %v", err)
		}
	}

	if conf.Address != "" {
		return nil, nil
	}
	if len(conf.SentinelAddresses) > 0 {
		return nil, nil
	}
	if len(conf.ClusterAddresses) > 0 {
		return nil, nil

	}

	return &conf, nil
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
		err = fmt.Errorf("could not connect to redis %v", err)
		return nil, err
	}

	return client, nil
}
