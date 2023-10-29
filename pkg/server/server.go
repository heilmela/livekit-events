package server

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"go.uber.org/zap"

	"github.com/heilmela/livekit-events/pkg/config"
	"github.com/livekit/protocol/auth"
	"github.com/livekit/protocol/livekit"
	"github.com/livekit/protocol/webhook"

	"github.com/gorilla/websocket"
)

type LivekitEventServer struct {
	logger       *zap.Logger
	authProvider *auth.SimpleKeyProvider
	wsUpgrader   *websocket.Upgrader
	config       *config.Config
}

func NewLivekitEventServer(logger *zap.Logger, conf *config.Config) *LivekitEventServer {
	return &LivekitEventServer{
		logger:       logger,
		authProvider: auth.NewSimpleKeyProvider(conf.LivekitConfig.ApiKey, conf.LivekitConfig.ApiSecret),
		config:       conf,
		wsUpgrader: &websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
	}
}

var clients = make(map[*websocket.Conn]bool)
var broadcast = make(chan *livekit.WebhookEvent)
var mutex = &sync.Mutex{}

func (s *LivekitEventServer) StartRedisPublisher() error {
	s.logger.Info("starting redis publisher")

	rdb, err := config.NewRedisClient(s.config.RedisConfig)
	if err != nil {
		return err
	}

	go func() {
		//lint:ignore S1000 reason
		for {
			select {
			case event, ok := <-broadcast:
				if !ok {
					return
				}
				jsonData, err := json.Marshal(event)
				if err != nil {
					s.logger.Sugar().Errorf("error converting event to JSON: %s", err)
				} else {
					_, err := rdb.Publish(context.Background(), s.config.RedisConfig.ChannelName, jsonData).Result()
					if err != nil {
						s.logger.Sugar().Errorf("failed to publish event to redis", err)
					}
				}
			}
		}
	}()

	return nil
}

func (s *LivekitEventServer) WebsocketHandler(w http.ResponseWriter, r *http.Request) {
	s.logger.Info("new client connected")

	ws, err := s.wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer ws.Close()

	mutex.Lock()
	clients[ws] = true
	mutex.Unlock()

	//lint:ignore S1000 reason
	for {
		select {
		case event, ok := <-broadcast:
			if !ok {
				return
			}
			for client := range clients {
				err := client.WriteJSON(event)
				if err != nil {
					s.logger.Sugar().Errorf("websocket error: %v", err)
					client.Close()
					mutex.Lock()
					delete(clients, client)
					mutex.Unlock()
				}
			}
		}
	}
}

func (s *LivekitEventServer) TrustUpstream(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var forwardedFor *string
		if value := r.Header.Get("X-Forwarded-For"); value != "" {
			forwardedFor = &value
		}
		remoteAddress := r.RemoteAddr

		for _, v := range s.config.ServerConfig.TrustedUpstream {
			if (forwardedFor != nil && *forwardedFor == v) || v == remoteAddress {
				next.ServeHTTP(w, r)
				return
			}
		}

		s.logger.Sugar().Warnf("untrusted event from %s %v", r.RemoteAddr, forwardedFor)
		http.Error(w, "Forbidden", http.StatusForbidden)
	})
}

func (s *LivekitEventServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var forwardedFor *string
	if value := r.Header.Get("X-Forwarded-For"); value != "" {
		forwardedFor = &value
	}

	s.logger.Sugar().Infof("received event from %s %v", r.RemoteAddr, forwardedFor)
	event, err := webhook.ReceiveWebhookEvent(r, s.authProvider)
	if err != nil {
		s.logger.Sugar().Errorf("event parsing faild", err)
		http.Error(w, "failed processing webhook event", http.StatusBadRequest)
		return
	}

	s.logger.Debug(event.String())

	broadcast <- event
	w.WriteHeader(http.StatusOK)
}
