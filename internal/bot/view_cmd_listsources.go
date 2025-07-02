package bot

import (
	"context"
	"fmt"
	"sort"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/ozaitsev92/gonewsbot/internal/botkit"
	"github.com/ozaitsev92/gonewsbot/internal/model"
)

type SourceLister interface {
	GetSources(ctx context.Context) ([]model.Source, error)
}

func ViewCmdListSource(lister SourceLister) botkit.ViewFunc {
	return func(ctx context.Context, bot *tgbotapi.BotAPI, update tgbotapi.Update) error {
		sources, err := lister.GetSources(ctx)
		if err != nil {
			return err
		}

		sort.SliceStable(sources, func(i, j int) bool {
			return sources[i].Priority > sources[j].Priority
		})

		sourceInfos := make([]string, len(sources))
		for i, source := range sources {
			sourceInfos[i] = formatSource(source)
		}

		msgText := fmt.Sprintf(
			"List of sources \\(total %d\\):\n\n%s",
			len(sources),
			strings.Join(sourceInfos, "\n\n"),
		)

		reply := tgbotapi.NewMessage(update.Message.Chat.ID, msgText)
		reply.ParseMode = parseModeMarkdownV2

		if _, err := bot.Send(reply); err != nil {
			return err
		}

		return nil
	}
}
