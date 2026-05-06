package destinations

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/agorischek/token-for-your-thoughts/internal/config"
	"github.com/agorischek/token-for-your-thoughts/internal/feedback"
)

type FileDestination struct {
	path   string
	format string
}

const markdownFeedbackPreamble = "# Feedback\n\nRuntime feedback entries collected by `tfyt` will appear here when the file destination is enabled."

func NewFileDestination(baseDir string, cfg config.DestinationConfig) (*FileDestination, error) {
	path := cfg.Path
	if !filepath.IsAbs(path) {
		path = filepath.Join(baseDir, path)
	}
	return &FileDestination{path: path, format: cfg.Format}, nil
}

func (s *FileDestination) Name() string {
	return "file"
}

func (s *FileDestination) Submit(_ context.Context, item feedback.Item) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	file, err := os.OpenFile(s.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open feedback file: %w", err)
	}
	defer file.Close()

	switch s.format {
	case "json":
		data, err := item.JSON(false)
		if err != nil {
			return fmt.Errorf("marshal feedback json: %w", err)
		}
		if _, err := file.Write(append(data, '\n')); err != nil {
			return fmt.Errorf("append json entry: %w", err)
		}
	default:
		existing, err := os.ReadFile(s.path)
		if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("read feedback file: %w", err)
		}

		content := strings.TrimRight(string(existing), "\n")
		entry := strings.TrimRight(item.MarkdownEntry(), "\n")
		if strings.TrimSpace(content) == "" {
			content = markdownFeedbackPreamble + "\n\n" + entry + "\n"
		} else {
			content = content + "\n\n" + entry + "\n"
		}

		if err := file.Truncate(0); err != nil {
			return fmt.Errorf("truncate feedback file: %w", err)
		}
		if _, err := file.Seek(0, 0); err != nil {
			return fmt.Errorf("seek feedback file: %w", err)
		}
		if _, err := file.WriteString(content); err != nil {
			return fmt.Errorf("write markdown entry: %w", err)
		}
	}

	return nil
}
