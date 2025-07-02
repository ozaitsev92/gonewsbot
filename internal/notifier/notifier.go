package notifier

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/go-shiori/go-readability"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/ozaitsev92/gonewsbot/internal/botkit/markup"
	"github.com/ozaitsev92/gonewsbot/internal/model"
)

type ArticlesProvider interface {
	AllNotPosted(ctx context.Context, since time.Time, limit uint64) ([]model.Article, error)
	MarkPosted(ctx context.Context, id int64) error
}

type Summarizer interface {
	Summarize(ctx context.Context, text string) (string, error)
}

type Notifier struct {
	articles         ArticlesProvider
	summarizer       Summarizer
	bot              *tgbotapi.BotAPI
	sendInterval     time.Duration
	lookupTimeWindow time.Duration
	channelID        int64
	httpClient       *http.Client
}

func NewNotifier(
	articles ArticlesProvider,
	summarizer Summarizer,
	bot *tgbotapi.BotAPI,
	sendInterval time.Duration,
	lookupTimeWindow time.Duration,
	channelID int64,
) *Notifier {
	// todo: get these values from config
	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			IdleConnTimeout:       90 * time.Second,
			MaxIdleConns:          100,
			MaxConnsPerHost:       100,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			DialContext: (&net.Dialer{
				Timeout:   5 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
		},
	}

	return &Notifier{
		articles:         articles,
		summarizer:       summarizer,
		bot:              bot,
		sendInterval:     sendInterval,
		lookupTimeWindow: lookupTimeWindow,
		channelID:        channelID,
		httpClient:       client,
	}
}

// todo: wrap the body of this method in a transaction
func (n *Notifier) SelectAndSendArticles(ctx context.Context) error {
	topOneArticles, err := n.articles.AllNotPosted(ctx, time.Now().Add(-n.lookupTimeWindow), 1)
	if err != nil {
		return err
	}

	if len(topOneArticles) == 0 {
		return nil
	}

	article := topOneArticles[0]

	summary, err := n.extractSummary(ctx, article)
	if err != nil {
		return err
	}

	if err := n.sendArticle(ctx, article, summary); err != nil {
		return err
	}

	if err := n.articles.MarkPosted(ctx, article.ID); err != nil {
		return err
	}

	return nil
}

func (n *Notifier) extractSummary(ctx context.Context, article model.Article) (string, error) {
	var r io.Reader

	if article.Summary != "" {
		r = strings.NewReader(article.Summary)
	} else {
		req, err := http.NewRequestWithContext(ctx, "GET", article.Link, nil)
		if err != nil {
			return "", err
		}

		resp, err := n.httpClient.Do(req)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("unexpected status code %d while fetching %s", resp.StatusCode, article.Link)
		}
		r = resp.Body
	}

	doc, err := readability.FromReader(r, nil)
	if err != nil {
		return "", err
	}

	summary, err := n.summarizer.Summarize(ctx, cleanText(doc.TextContent))
	if err != nil {
		return "", err
	}

	return "\n\n" + summary, nil
}

func (n *Notifier) sendArticle(ctx context.Context, article model.Article, summary string) error {
	const msgFormat = "*%s*%s\n\n%s"

	msg := tgbotapi.NewMessage(
		n.channelID,
		fmt.Sprintf(
			msgFormat,
			markup.EscapeForMarkdown(article.Title),
			markup.EscapeForMarkdown(summary),
			article.Link,
		),
	)
	msg.ParseMode = tgbotapi.ModeMarkdownV2

	_, err := n.bot.Send(msg)
	if err != nil {
		return err
	}

	return nil
}

func cleanText(text string) string {
	return regexp.MustCompile(`\n{3,}`).ReplaceAllString(strings.TrimSpace(text), "\n")
}
