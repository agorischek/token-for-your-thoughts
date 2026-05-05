package sinks

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/agorischek/suggestion-box/internal/config"
	"github.com/agorischek/suggestion-box/internal/feedback"
	_ "modernc.org/sqlite"
)

var identifierPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

type SQLSink struct {
	db         *sql.DB
	driver     string
	insertStmt string
}

func NewSQLSink(baseDir string, cfg config.SinkConfig) (*SQLSink, error) {
	driver := strings.TrimSpace(cfg.Driver)
	dsn := strings.TrimSpace(cfg.DSN)
	if driver == "" || dsn == "" {
		return nil, fmt.Errorf("sql sink requires driver and dsn")
	}

	if isSQLiteDriver(driver) && shouldResolvePath(dsn) {
		dsn = filepath.Join(baseDir, dsn)
	}

	db, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, fmt.Errorf("open sql database: %w", err)
	}

	createStmt, insertStmt, err := sqlStatements(cfg)
	if err != nil {
		return nil, err
	}

	if cfg.AutoCreate != nil && *cfg.AutoCreate && createStmt != "" {
		if _, err := db.Exec(createStmt); err != nil {
			return nil, fmt.Errorf("create sql table: %w", err)
		}
	}

	return &SQLSink{
		db:         db,
		driver:     driver,
		insertStmt: insertStmt,
	}, nil
}

func (s *SQLSink) Name() string {
	return "sql"
}

func (s *SQLSink) Submit(ctx context.Context, item feedback.Item) error {
	if _, err := s.db.ExecContext(ctx, s.insertStmt,
		item.ID,
		item.Provider,
		item.Summary,
		item.Details,
		item.Category,
		item.Source,
		item.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		item.MetadataJSON(),
	); err != nil {
		return fmt.Errorf("insert feedback row: %w", err)
	}
	return nil
}

func sqlStatements(cfg config.SinkConfig) (string, string, error) {
	if strings.TrimSpace(cfg.InsertStmt) != "" {
		return cfg.CreateStmt, cfg.InsertStmt, nil
	}

	if !identifierPattern.MatchString(cfg.Table) {
		return "", "", fmt.Errorf("sql sink table must be a simple identifier unless insert_statement is provided")
	}

	switch {
	case isSQLiteDriver(cfg.Driver):
		create := cfg.CreateStmt
		if strings.TrimSpace(create) == "" {
			create = fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
	id TEXT PRIMARY KEY,
	provider TEXT NOT NULL,
	summary TEXT NOT NULL,
	details TEXT NOT NULL,
	category TEXT NOT NULL,
	source TEXT NOT NULL,
	created_at TEXT NOT NULL,
	metadata_json TEXT NOT NULL
);`, cfg.Table)
		}
		insert := fmt.Sprintf(`INSERT INTO %s (id, provider, summary, details, category, source, created_at, metadata_json)
VALUES (?, ?, ?, ?, ?, ?, ?, ?);`, cfg.Table)
		return create, insert, nil
	default:
		return "", "", fmt.Errorf("sql sink for driver %q requires insert_statement", cfg.Driver)
	}
}

func isSQLiteDriver(driver string) bool {
	return driver == "sqlite" || driver == "sqlite3"
}

func shouldResolvePath(dsn string) bool {
	if dsn == ":memory:" {
		return false
	}
	return !strings.HasPrefix(dsn, "file:") && !filepath.IsAbs(dsn)
}
