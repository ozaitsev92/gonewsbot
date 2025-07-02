package fetcher

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/ozaitsev92/gonewsbot/internal/model"
	"github.com/ozaitsev92/gonewsbot/internal/source"
)

type ArticleStorage interface {
	AddArticle(ctx context.Context, article model.Article) (int64, error)
}

type SourceProvider interface {
	GetSources(ctx context.Context) ([]model.Source, error)
}

type Source interface {
	ID() int64
	Name() string
	Fetch(ctx context.Context) ([]model.Item, error)
}

type Fetcher struct {
	articles ArticleStorage
	sources  SourceProvider

	fetchInterval  time.Duration
	filterKeywords []string
}

func New(
	articles ArticleStorage,
	sources SourceProvider,
	fetchInterval time.Duration,
	filterKeywords []string,
) *Fetcher {
	return &Fetcher{
		articles:       articles,
		sources:        sources,
		fetchInterval:  fetchInterval,
		filterKeywords: filterKeywords,
	}
}
func (f *Fetcher) Start(ctx context.Context) error {
	ticker := time.NewTicker(f.fetchInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := f.Fetch(ctx); err != nil {
				return err
			}
		}
	}
}

func (f *Fetcher) Fetch(ctx context.Context) error {
	sources, err := f.sources.GetSources(ctx)
	if err != nil {
		return err
	}

	var wg sync.WaitGroup

	for _, src := range sources {
		wg.Add(1)

		rssSource := source.NewRSSSourceFromModel(src)

		go func(source Source) {
			defer wg.Done()

			items, err := source.Fetch(ctx)
			if err != nil {
				slog.Error("failed to fetch items", "source", source.Name(), "error", err)
				return
			}

			if err := f.processItems(ctx, source, items); err != nil {
				slog.Error("failed to process items", "source", source.Name(), "error", err)
				return
			}
		}(rssSource)
	}

	wg.Wait()

	return nil
}

func (f *Fetcher) processItems(ctx context.Context, source Source, items []model.Item) error {
	for _, item := range items {
		if f.itemShouldBeSkipped(item) {
			continue
		}

		article := model.Article{
			SourceID:    source.ID(),
			Title:       item.Title,
			Link:        item.Link,
			Summary:     item.Summary,
			PublishedAt: item.Date,
		}

		if _, err := f.articles.AddArticle(ctx, article); err != nil {
			return err
		}
	}

	return nil
}

func (f *Fetcher) itemShouldBeSkipped(item model.Item) bool {
	categories := make(map[string]struct{}, len(item.Categories))
	for _, category := range item.Categories {
		categories[category] = struct{}{}
	}

	title := strings.ToLower(item.Title)
	for _, keyword := range f.filterKeywords {
		if _, exists := categories[keyword]; exists || strings.Contains(title, keyword) {
			return true
		}
	}

	return false
}
