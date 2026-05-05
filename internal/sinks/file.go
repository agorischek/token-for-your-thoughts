package sinks

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/agorischek/suggestion-box/internal/config"
	"github.com/agorischek/suggestion-box/internal/feedback"
)

type FileSink struct {
	path string
}

func NewFileSink(baseDir string, cfg config.SinkConfig) (*FileSink, error) {
	path := cfg.Path
	if !filepath.IsAbs(path) {
		path = filepath.Join(baseDir, path)
	}
	return &FileSink{path: path}, nil
}

func (s *FileSink) Name() string {
	return "file"
}

func (s *FileSink) Submit(_ context.Context, item feedback.Item) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	file, err := os.OpenFile(s.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open feedback file: %w", err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("stat feedback file: %w", err)
	}
	if info.Size() == 0 {
		if _, err := file.WriteString("# Feedback\n\n"); err != nil {
			return fmt.Errorf("write header: %w", err)
		}
	}

	if _, err := file.WriteString(item.MarkdownEntry()); err != nil {
		return fmt.Errorf("append entry: %w", err)
	}

	return nil
}
