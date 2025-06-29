package source

import (
	"context"

	"github.com/SlyMarbo/rss"
	"github.com/ozaitsev92/gonewsbot/internal/model"
)

type response struct {
	feed *rss.Feed
	err  error
}

type RSSSource struct {
	URL        string
	SourceID   int64
	SourceName string
}

func NewRSSSourceFromModel(m model.Source) RSSSource {
	return RSSSource{
		URL:        m.FeedURL,
		SourceID:   m.ID,
		SourceName: m.Name,
	}
}

func (s RSSSource) Fetch(ctx context.Context) ([]model.Item, error) {
	feed, err := s.loadFeed(ctx, s.URL)
	if err != nil {
		return nil, err
	}

	items := make([]model.Item, len(feed.Items))
	for i, item := range feed.Items {
		items[i] = model.Item{
			Title:      item.Title,
			Categories: item.Categories,
			Link:       item.Link,
			Date:       item.Date,
			Summary:    item.Summary,
			SourceName: s.SourceName,
		}
	}

	return items, nil
}

func (s RSSSource) loadFeed(ctx context.Context, url string) (*rss.Feed, error) {
	resCh := make(chan response)

	go func() {
		feed, err := rss.Fetch(url)
		res := response{
			feed: feed,
			err:  err,
		}
		resCh <- res
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case res := <-resCh:
		return res.feed, res.err
	}
}

func (s RSSSource) ID() int64 {
	return s.SourceID
}

func (s RSSSource) Name() string {
	return s.SourceName
}
