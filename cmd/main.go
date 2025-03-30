package main

import (
	"context"
	"github.com/jaam8/mattermost_bot/internal/api"
	"github.com/jaam8/mattermost_bot/internal/config"
	"github.com/jaam8/mattermost_bot/internal/repository"
	srv "github.com/jaam8/mattermost_bot/internal/service"
	"github.com/jaam8/mattermost_bot/pkg/logger"
	"github.com/jaam8/mattermost_bot/pkg/tarantool"
	"github.com/mattermost/mattermost-server/v6/model"
	got "github.com/tarantool/go-tarantool"
	"go.uber.org/zap"
	logg "log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	ctx := context.Background()
	ctx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	cfg, err := config.New()
	if err != nil {
		logg.Fatalf("failed to load config: %s", err)
	}
	log, err := logger.New(cfg.LogLevel)
	if err != nil {
		logg.Fatalf("failed to initalize logger: %s", err)
	}
	conn, err := tarantool.New(cfg.Tarantool)
	if err != nil {
		logg.Fatalf("failed to connect to Tarantool: %s", err)
	}

	client := model.NewAPIv4Client(cfg.MmURL)
	webSocketClient, err := model.NewWebSocketClient4(cfg.MmWsURL, cfg.BotToken)
	if err != nil {
		logg.Fatalf("failed to connect to webSocket: %v", err)
	}

	repo := repository.New(conn, log)
	service := srv.New(repo, log)
	handler := api.New(service, log, client, cfg.ChannelID)

	client.SetToken(cfg.BotToken)
	webSocketClient.Listen()
	var botID string
	if user, _, err := client.GetUser("me", ""); err != nil {
		logg.Fatalf("failed to get user: %s", err)

	} else {
		botID = user.Id
	}

	go func() {
		for event := range webSocketClient.EventChannel {
			if event.EventType() == model.WebsocketEventPosted {
				log.Debug("new message", zap.String("event", event.EventType()))
				api.HandleMessage(handler, event, botID)
			}
		}
	}()

	select {
	case <-ctx.Done():
		resp, err := conn.Select("polls", "primary", 0, 10, got.IterEq, []interface{}{})
		if err != nil {
			logg.Fatalf("failed to select: %s", err)
		}
		logg.Println(resp)
		conn.CloseGraceful()
		webSocketClient.Close()
		stop()
		logg.Println("server graceful stopped")
	}
}
