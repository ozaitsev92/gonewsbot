package storage

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/ozaitsev92/gonewsbot/internal/model"
)

type dbSource struct {
	ID        int64     `db:"id"`
	Name      string    `db:"name"`
	FeedURL   string    `db:"feed_url"`
	CreatedAt time.Time `db:"created_at"`
}

type SourcePostgresStorage struct {
	db *sqlx.DB
}

func NewSourcePostgresStorage(db *sqlx.DB) *SourcePostgresStorage {
	return &SourcePostgresStorage{
		db: db,
	}
}

func (s *SourcePostgresStorage) GetSources(ctx context.Context) ([]model.Source, error) {
	conn, err := s.db.Conn(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	var sources []dbSource
	rows, err := conn.QueryContext(ctx, "SELECT * FROM sources")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var src dbSource
		if err := rows.Scan(&src.ID, &src.Name, &src.FeedURL, &src.CreatedAt); err != nil {
			return nil, err
		}
		sources = append(sources, src)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	result := make([]model.Source, len(sources))
	for i, src := range sources {
		result[i] = model.Source(src)
	}

	return result, nil
}

func (s *SourcePostgresStorage) GetSourceByID(ctx context.Context, id int64) (*model.Source, error) {
	conn, err := s.db.Conn(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	var src dbSource
	row := conn.QueryRowContext(ctx, "SELECT * FROM sources WHERE id = $1", id)
	if err := row.Err(); err != nil {
		return nil, err
	}

	if err := row.Scan(&src); err != nil {
		return nil, err
	}

	result := model.Source(src)
	return &result, nil
}

func (s *SourcePostgresStorage) AddSource(ctx context.Context, source model.Source) (int64, error) {
	conn, err := s.db.Conn(ctx)
	if err != nil {
		return 0, err
	}
	defer conn.Close()

	row := conn.QueryRowContext(
		ctx,
		"INSERT INTO sources (name, feed_url, created_at) VALUES ($1, $2, $3) RETURNING id",
		source.Name,
		source.FeedURL,
		source.CreatedAt,
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

func (s *SourcePostgresStorage) DeleteSource(ctx context.Context, id int64) error {
	conn, err := s.db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	_, err = conn.ExecContext(ctx, "DELETE FROM sources WHERE id = $1", id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}

		return err
	}

	return nil
}
