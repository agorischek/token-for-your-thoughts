package sinks

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/agorischek/token-for-your-thoughts/internal/config"
	"github.com/agorischek/token-for-your-thoughts/internal/feedback"
)

type SQLSink struct {
	db         *sql.DB
	insertStmt string
}

func NewSQLSink(_ string, cfg config.SinkConfig) (*SQLSink, error) {
	driver := strings.TrimSpace(cfg.Driver)
	dsn := strings.TrimSpace(cfg.DSN)
	if driver == "" || dsn == "" {
		return nil, fmt.Errorf("sql sink requires driver and dsn")
	}

	db, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, fmt.Errorf("open sql database: %w", err)
	}

	if cfg.AutoCreate != nil && *cfg.AutoCreate {
		if _, err := db.Exec(cfg.CreateStmt); err != nil {
			return nil, fmt.Errorf("create sql table: %w", err)
		}
	}

	return &SQLSink{
		db:         db,
		insertStmt: cfg.InsertStmt,
	}, nil
}

func (s *SQLSink) Name() string {
	return "sql"
}

func (s *SQLSink) Close() error {
	return s.db.Close()
}

func (s *SQLSink) Submit(ctx context.Context, item feedback.Item) error {
	if _, err := s.db.ExecContext(ctx, s.insertStmt,
		item.ID,
		item.Provider,
		item.Feedback,
		item.Source,
		item.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		item.MetadataJSON(),
	); err != nil {
		return fmt.Errorf("insert feedback row: %w", err)
	}
	return nil
}
