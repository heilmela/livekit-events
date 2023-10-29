package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gorilla/mux"
	log "github.com/heilmela/livekit-events/internal"
	cfg "github.com/heilmela/livekit-events/pkg/config"
	server "github.com/heilmela/livekit-events/pkg/server"

	_ "go.uber.org/automaxprocs"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "an error occurred: %s\n", err)
		os.Exit(1)
	}
}

func run() error {
	configPath := flag.String("config", "", "Path to config file")
	apiKey := flag.String("key", "", "Livekit api key")
	apiSecret := flag.String("secret", "", "Livekit api secret")
	flag.Parse()

	path := *configPath
	if path == "" {
		path = "./config.yaml"
	}
	path, err := filepath.Abs("./config.yaml")

	if err == nil {
		_, err := os.Stat(path)
		if err != nil {
			path = ""
		}
	} else {
		path = ""
	}

	config, cfgErr := cfg.NewConfig(path)
	if cfgErr != nil && config == nil {
		return cfgErr
	}

	logger, err := log.NewAtLevel(config.LogLevel)

	if err != nil {
		fmt.Println(err)
	}

	defer func() {
		err = logger.Sync()
	}()

	if cfgErr != nil {
		logger.Sugar().Errorf("config parsing error %v", cfgErr)
	}

	if *apiKey != "" {
		config.LivekitConfig.ApiKey = *apiKey
	}

	if *apiSecret != "" {
		config.LivekitConfig.ApiSecret = *apiSecret
	}

	router := mux.NewRouter()
	srv := server.NewLivekitEventServer(
		logger,
		config,
	)
	router.HandleFunc("/", srv.WebsocketHandler)

	handler := srv.ServeHTTP

	if len(config.ServerConfig.TrustedUpstream) > 0 {
		wrappedHandler := srv.TrustUpstream(http.HandlerFunc(handler))
		router.HandleFunc("/webhook", wrappedHandler.ServeHTTP).Methods("POST")
	} else {
		router.HandleFunc("/webhook", handler).Methods("POST")
	}
	if config.RedisConfig != nil {
		if err := srv.StartRedisPublisher(); err != nil {
			logger.Error("failed to start redis publisher")
			return err
		}
	}

	logger.Sugar().Infof("started on %s:%v", config.ServerConfig.BindAddresse, config.ServerConfig.Port)
	err = http.ListenAndServe(fmt.Sprintf("%s:%v", config.ServerConfig.BindAddresse, config.ServerConfig.Port), router)

	return err
}
