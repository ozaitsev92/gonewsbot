package storage

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/ozaitsev92/gonewsbot/internal/model"
)

type dbArticle struct {
	ID          int64     `db:"id"`
	SourceID    int64     `db:"source_id"`
	Title       string    `db:"title"`
	Link        string    `db:"link"`
	Summary     string    `db:"summary"`
	PublishedAt time.Time `db:"published_at"`
	PostedAt    time.Time `db:"posted_at"`
	CreatedAt   time.Time `db:"created_at"`
}

type ArticlePostgresStorage struct {
	db *sqlx.DB
}

func NewArticlePostgresStorage(db *sqlx.DB) *ArticlePostgresStorage {
	return &ArticlePostgresStorage{
		db: db,
	}
}

func (s *ArticlePostgresStorage) AddArticle(ctx context.Context, article model.Article) (int64, error) {
	conn, err := s.db.Conn(ctx)
	if err != nil {
		return 0, err
	}
	defer conn.Close()

	row := conn.QueryRowContext(
		ctx,
		`
			INSERT INTO articles (source_id, title, link, summary, published_at)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT DO NOTHING
			RETURNING id
		`,
		article.SourceID,
		article.Title,
		article.Link,
		article.Summary,
		article.PublishedAt,
	)
	if err := row.Err(); err != nil {
		return 0, err
	}

	var id int64
	if err := row.Scan(&id); err != nil {
		return 0, err
	}

	return id, nil
}

func (s *ArticlePostgresStorage) AllNotPosted(ctx context.Context, since time.Time, limit int) ([]model.Article, error) {
	conn, err := s.db.Conn(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	var articles []dbArticle
	rows, err := conn.QueryContext(
		ctx,
		`
			SELECT * FROM articles
			WHERE posted_at IS NULL AND published_at >= $1::timestamp
			ORDER BY published_at DESC
			LIMIT $2
		`,
		since.UTC().Format(time.RFC3339),
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var src dbArticle
		if err := rows.Scan(&src.ID, &src.SourceID, &src.Title, &src.Link, &src.Summary, &src.PublishedAt, &src.PostedAt, &src.CreatedAt); err != nil {
			return nil, err
		}
		articles = append(articles, src)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	result := make([]model.Article, len(articles))
	for i, src := range articles {
		result[i] = model.Article(src)
	}

	return result, nil
}

func (s *ArticlePostgresStorage) MarkPosted(ctx context.Context, id int64) error {
	conn, err := s.db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	_, err = conn.ExecContext(
		ctx,
		"UPDATE articles SET posted_at = $1::timestamp WHERE id = $1",
		id,
		time.Now().UTC().Format(time.RFC3339),
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}

		return err
	}

	return nil
}
