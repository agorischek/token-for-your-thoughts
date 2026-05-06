package destinations

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/agorischek/token-for-your-thoughts/internal/config"
	"github.com/agorischek/token-for-your-thoughts/internal/feedback"
)

type SQLDestination struct {
	db         *sql.DB
	insertStmt string
}

func NewSQLDestination(_ string, cfg config.DestinationConfig) (*SQLDestination, error) {
	driver := strings.TrimSpace(cfg.Driver)
	dsn := strings.TrimSpace(cfg.DSN)
	if driver == "" || dsn == "" {
		return nil, fmt.Errorf("sql destination requires driver and dsn")
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

	return &SQLDestination{
		db:         db,
		insertStmt: cfg.InsertStmt,
	}, nil
}

func (s *SQLDestination) Name() string {
	return "sql"
}

func (s *SQLDestination) Close(_ context.Context) error {
	return s.db.Close()
}

// Submit executes the configured insert statement with positional parameters
// in the following fixed order:
//  1. ID (string)
//  2. Provider (string)
//  3. Feedback (string)
//  4. Source (string)
//  5. CreatedAt (RFC 3339 timestamp string)
//  6. Metadata (JSON string)
func (s *SQLDestination) Submit(ctx context.Context, item feedback.Item) error {
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
