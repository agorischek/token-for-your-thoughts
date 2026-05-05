package sinks

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/agorischek/suggesting/internal/config"
	"github.com/agorischek/suggesting/internal/feedback"
)

func TestGitSinkWritesPerFeedbackFile(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	sink, err := NewGitSink(repoRoot, repoRoot, config.SinkConfig{
		Type:      "git",
		Directory: ".feedback",
		Branch:    "feedback",
		Remote:    "origin",
		Push:      boolPtr(false),
	})
	if err != nil {
		t.Fatalf("new git sink: %v", err)
	}

	item, err := feedback.New("Claude Code", "The CLI hid the real error output.", "cli", nil)
	if err != nil {
		t.Fatalf("new item: %v", err)
	}

	if filepath.IsAbs(sink.directory) {
		t.Fatal("expected relative directory")
	}

	path := filepath.Join(repoRoot, sink.directory, item.ID+".md")
	if rel, err := filepath.Rel(repoRoot, filepath.Join(repoRoot, filepath.Clean(sink.directory))); err != nil || strings.HasPrefix(rel, "..") {
		t.Fatalf("directory escaped repo: rel=%q err=%v", rel, err)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(item.MarkdownEntry()), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	for _, expected := range []string{item.ID, "Provider: Claude Code", "Feedback: The CLI hid the real error output."} {
		if !strings.Contains(string(data), expected) {
			t.Fatalf("file missing %q", expected)
		}
	}
}

func boolPtr(v bool) *bool {
	return &v
}
