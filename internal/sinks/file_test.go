package sinks

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/agorischek/token-for-your-thoughts/internal/config"
	"github.com/agorischek/token-for-your-thoughts/internal/feedback"
)

func TestFileSinkWritesEntry(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	sink, err := NewFileSink(dir, config.SinkConfig{Type: "file", Path: "FEEDBACK.md"})
	if err != nil {
		t.Fatalf("new file sink: %v", err)
	}

	item, err := feedback.New("Claude Code", "Summary and details together", "cli", nil)
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
	for _, expected := range []string{"# Feedback", "Claude Code", "Summary and details together"} {
		if !strings.Contains(string(data), expected) {
			t.Fatalf("file missing %q", expected)
		}
	}
}

func TestFileSinkWritesJSONEntry(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	sink, err := NewFileSink(dir, config.SinkConfig{Type: "file", Path: "FEEDBACK.jsonl", Format: "json"})
	if err != nil {
		t.Fatalf("new file sink: %v", err)
	}

	item, err := feedback.New("Claude Code", "JSON output should be machine-friendly.", "cli", map[string]any{"key": "value"})
	if err != nil {
		t.Fatalf("new item: %v", err)
	}
	if err := sink.Submit(context.Background(), item); err != nil {
		t.Fatalf("submit: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "FEEDBACK.jsonl"))
	if err != nil {
		t.Fatalf("read file: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected 1 json line, got %d", len(lines))
	}

	var decoded feedback.Item
	if err := json.Unmarshal([]byte(lines[0]), &decoded); err != nil {
		t.Fatalf("unmarshal json line: %v", err)
	}
	if decoded.ID != item.ID || decoded.Feedback != item.Feedback {
		t.Fatalf("decoded item mismatch: %#v", decoded)
	}
}
