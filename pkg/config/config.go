package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v2"
)

type Config struct {
	LogLevel      string         `yaml:"log_level,omitempty"`
	RedisConfig   *RedisConfig   `yaml:"redis,omitempty"`
	ServerConfig  *ServerConfig  `yaml:"server,omitempty"`
	LivekitConfig *LivekitConfig `yaml:"livekit,omitempty"`
}

type ServerConfig struct {
	Port            uint32   `yaml:"port,omitempty"`
	BindAddresse    string   `yaml:"bind_addresse,omitempty"`
	TrustedUpstream []string `yaml:"trusted_upstreams,omitempty"`
}

type LivekitConfig struct {
	ApiKey    string `yaml:"api_key,omitempty"`
	ApiSecret string `yaml:"api_secret,omitempty"`
}

var DefaultConfig = Config{
	ServerConfig: &ServerConfig{
		Port:         3000,
		BindAddresse: "0.0.0.0",
	},
	LogLevel: "debug",
}

func NewConfig(path string) (*Config, error) {
	marshalled, err := yaml.Marshal(&DefaultConfig)
	if err != nil {
		return nil, err
	}

	var conf Config
	err = yaml.Unmarshal(marshalled, &conf)
	if err != nil {
		return nil, err
	}

	if value, ok := os.LookupEnv("PORT"); ok {
		if port, err := strconv.ParseInt(value, 10, 32); err == nil {
			conf.ServerConfig.Port = uint32(port)
		}
	}

	if value, ok := os.LookupEnv("HOST"); ok {
		conf.ServerConfig.BindAddresse = value
	}

	if value, ok := os.LookupEnv("LIVEKIT_API_SECRET"); ok {
		conf.LivekitConfig.ApiSecret = value
	}

	if value, ok := os.LookupEnv("LIVEKIT_API_KEY"); ok {
		conf.LivekitConfig.ApiKey = value
	}
	if value, ok := os.LookupEnv("TRUSTED_UPSTREAMS"); ok {
		conf.ServerConfig.TrustedUpstream = strings.Split(value, ",")
	}

	if path != "" {
		bytes, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		confString := string(bytes)
		decoder := yaml.NewDecoder(strings.NewReader(confString))

		if err := decoder.Decode(&conf); err != nil {
			return nil, fmt.Errorf("could not parse config: %v", err)
		}
	}

	rcConf, err := NewRedisConfig(path)

	if rcConf != nil {
		conf.RedisConfig = rcConf
	}

	fmt.Println(conf)

	return &conf, err
}
