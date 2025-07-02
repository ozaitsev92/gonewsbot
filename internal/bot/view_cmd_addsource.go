package bot

import (
	"context"
	"fmt"
	"log/slog"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/ozaitsev92/gonewsbot/internal/botkit"
	"github.com/ozaitsev92/gonewsbot/internal/model"
)

type SourceStorage interface {
	AddSource(ctx context.Context, source model.Source) (int64, error)
}

func ViewCmdAddSource(storage SourceStorage) botkit.ViewFunc {
	type addSourceArgs struct {
		Name     string `json:"name"`
		URL      string `json:"url"`
		Priority int    `json:"priority"`
	}

	return func(ctx context.Context, bot *tgbotapi.BotAPI, update tgbotapi.Update) error {
		args, err := botkit.ParseJSON[addSourceArgs](update.Message.CommandArguments())
		if err != nil {
			return err
		}

		source := model.Source{
			Name:     args.Name,
			FeedURL:  args.URL,
			Priority: args.Priority,
		}

		sourceID, err := storage.AddSource(ctx, source)
		if err != nil {
			slog.Error("failed to add source", "error", err)
			return err
		}

		msgText := fmt.Sprintf(
			"Source added with ID: `%d`\\. Use this ID to update or delete the source\\.",
			sourceID,
		)

		reply := tgbotapi.NewMessage(update.Message.Chat.ID, msgText)

		reply.ParseMode = parseModeMarkdownV2

		if _, err := bot.Send(reply); err != nil {
			return err
		}

		return nil
	}
}
