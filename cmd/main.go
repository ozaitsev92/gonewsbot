package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/jmoiron/sqlx"
	"github.com/ozaitsev92/gonewsbot/internal/config"
	"github.com/ozaitsev92/gonewsbot/internal/fetcher"
	"github.com/ozaitsev92/gonewsbot/internal/notifier"
	"github.com/ozaitsev92/gonewsbot/internal/storage"
	"github.com/ozaitsev92/gonewsbot/internal/summary"
)

func main() {
	botAPI, err := tgbotapi.NewBotAPI(config.Get().TelegramBotToken)
	if err != nil {
		slog.Error("failed to create bot API", "error", err)
		return
	}

	db, err := sqlx.Connect("postgres", config.Get().DatabaseDSN)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		return
	}
	defer db.Close()

	articlesStorage := storage.NewArticlePostgresStorage(db)

	sourcesStorage := storage.NewSourcePostgresStorage(db)

	aFetcher := fetcher.New(
		articlesStorage,
		sourcesStorage,
		config.Get().FetchInterval,
		config.Get().FilterKeywords,
	)

	aNotifier := notifier.NewNotifier(
		articlesStorage,
		summary.NewOpenAISummarizer(
			config.Get().OpenAIKey,
			config.Get().OpenAIPrompt,
		),
		botAPI,
		config.Get().NotificationInterval,
		2*config.Get().FetchInterval,
		config.Get().TelegramChannelID,
	)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	go func(ctx context.Context) {
		if err := aFetcher.Start(ctx); err != nil {
			if !errors.Is(err, context.Canceled) {
				slog.Error("fetcher stopped with error", "error", err)
			} else {
				slog.Info("fetcher stopped due to context cancellation")
			}
		}
	}(ctx)

	go func(ctx context.Context) {
		if err := aNotifier.Start(ctx); err != nil {
			if !errors.Is(err, context.Canceled) {
				slog.Error("notifier stopped with error", "error", err)
			} else {
				slog.Info("notifier stopped due to context cancellation")
			}
		}
	}(ctx)

	<-ctx.Done()
	slog.Info("shutting down gracefully")

	if err := db.Close(); err != nil {
		slog.Error("failed to close database connection", "error", err)
	}

	slog.Info("shutdown complete")
}
