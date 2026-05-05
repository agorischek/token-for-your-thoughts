package sinks

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/agorischek/suggestion-box/internal/config"
	"github.com/agorischek/suggestion-box/internal/feedback"
)

func TestFileSinkWritesEntry(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	sink, err := NewFileSink(dir, config.SinkConfig{Type: "file", Path: "FEEDBACK.md"})
	if err != nil {
		t.Fatalf("new file sink: %v", err)
	}

	item, err := feedback.New("Claude Code", "Summary", "Details", "tooling", "cli", nil)
	if err != nil {
		t.Fatalf("new item: %v", err)
	}
	if err := sink.Submit(context.Background(), item); err != nil {
		t.Fatalf("submit: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "FEEDBACK.md"))
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	for _, expected := range []string{"# Feedback", "Claude Code", "Summary"} {
		if !strings.Contains(string(data), expected) {
			t.Fatalf("file missing %q", expected)
		}
	}
}
