package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type Config struct {
	TgAPIKey string `json:"tgAPIKey"`
	ProxyURL string `json:"proxyURL"`
}

func main() {
	data, err := os.ReadFile("/home/feb/.config/tau/server.json")
	if err != nil {
		panic(err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		panic(err)
	}

	if cfg.TgAPIKey == "" {
		panic("tgAPIKey is empty in config")
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	if cfg.ProxyURL != "" {
		proxyURL, err := url.Parse(cfg.ProxyURL)
		if err != nil {
			panic(err)
		}
		httpClient.Transport = &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		}
	}

	opts := []bot.Option{
		bot.WithHTTPClient(time.Second*10, httpClient),
		bot.WithDefaultHandler(handler),
	}

	b, err := bot.New(cfg.TgAPIKey, opts...)
	if err != nil {
		panic(err)
	}

	b.Start(ctx)
}

func handler(ctx context.Context, b *bot.Bot, update *models.Update) {
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   update.Message.Text,
	})
}
