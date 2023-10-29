package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	log "github.com/heilmela/livekit-events/internal"
	cfg "github.com/heilmela/livekit-events/pkg/config"
	server "github.com/heilmela/livekit-events/pkg/server"
	"gopkg.in/yaml.v2"

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
	flag.Parse()

	config, err := cfg.NewConfig(*configPath)
	if err != nil && config == nil {
		return err
	}

	logger, err := log.NewAtLevel(config.LogLevel)
	if err != nil {
		return err
	}
	defer func() {
		err = logger.Sync()
	}()

	yamlConfig, err := yaml.Marshal(config)
	if err != nil {
		return err
	}
	logger.Debug(string(yamlConfig))

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

	server := &http.Server{
		Addr:              fmt.Sprintf("%s:%v", config.ServerConfig.BindAddresse, config.ServerConfig.Port),
		ReadHeaderTimeout: 5 * time.Second,
		Handler:           router,
	}

	logger.Sugar().Infof("started on %s:%v", config.ServerConfig.BindAddresse, config.ServerConfig.Port)
	err = server.ListenAndServe()

	return err
}
