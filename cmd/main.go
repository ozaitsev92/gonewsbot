package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/jmoiron/sqlx"
	"github.com/ozaitsev92/gonewsbot/internal/bot"
	"github.com/ozaitsev92/gonewsbot/internal/bot/middleware"
	"github.com/ozaitsev92/gonewsbot/internal/botkit"
	"github.com/ozaitsev92/gonewsbot/internal/config"
	"github.com/ozaitsev92/gonewsbot/internal/fetcher"
	"github.com/ozaitsev92/gonewsbot/internal/notifier"
	"github.com/ozaitsev92/gonewsbot/internal/storage"
	"github.com/ozaitsev92/gonewsbot/internal/summary"
)

func main() {
	cfg := config.Get()

	botAPI, err := tgbotapi.NewBotAPI(cfg.TelegramBotToken)
	if err != nil {
		slog.Error("failed to create bot API", "error", err)
		return
	}

	db, err := sqlx.Connect("postgres", cfg.DatabaseDSN)
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
		cfg.FetchInterval,
		cfg.FilterKeywords,
	)

	aNotifier := notifier.NewNotifier(
		articlesStorage,
		summary.NewOpenAISummarizer(
			config.Get().OpenAIKey,
			config.Get().OpenAIModel,
			config.Get().OpenAIPrompt,
		),
		botAPI,
		cfg.NotificationInterval,
		2*cfg.FetchInterval,
		cfg.TelegramChannelID,
	)

	newsBot := botkit.New(botAPI)
	newsBot.RegisterCmdView("addsource", middleware.AdminsOnly(cfg.TelegramChannelID, bot.ViewCmdAddSource(sourcesStorage)))
	newsBot.RegisterCmdView("setpriority", middleware.AdminsOnly(cfg.TelegramChannelID, bot.ViewCmdSetPriority(sourcesStorage)))
	newsBot.RegisterCmdView("getsource", middleware.AdminsOnly(cfg.TelegramChannelID, bot.ViewCmdGetSource(sourcesStorage)))
	newsBot.RegisterCmdView("listsources", middleware.AdminsOnly(cfg.TelegramChannelID, bot.ViewCmdListSource(sourcesStorage)))
	newsBot.RegisterCmdView("deletesource", middleware.AdminsOnly(cfg.TelegramChannelID, bot.ViewCmdDeleteSource(sourcesStorage)))

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	server := &http.Server{
		Addr:    cfg.HTTPBindAddress,
		Handler: mux,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Start fetcher
	go func() {
		if err := aFetcher.Start(ctx); err != nil && !errors.Is(err, context.Canceled) {
			slog.Error("fetcher stopped with error", "error", err)
		}
	}()

	// Start notifier
	go func() {
		if err := aNotifier.Start(ctx); err != nil && !errors.Is(err, context.Canceled) {
			slog.Error("notifier stopped with error", "error", err)
		}
	}()

	// Start HTTP server
	go func() {
		slog.Info("starting HTTP server", "addr", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("HTTP server error", "error", err)
		}
	}()

	// Run Telegram bot (blocking)
	if err := newsBot.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		slog.Error("bot stopped with error", "error", err)
	}

	// Graceful shutdown of HTTP server
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("failed to shutdown HTTP server gracefully", "error", err)
	} else {
		slog.Info("HTTP server shutdown complete")
	}
}
