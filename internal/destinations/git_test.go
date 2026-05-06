package destinations

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/agorischek/token-for-your-thoughts/internal/config"
	"github.com/agorischek/token-for-your-thoughts/internal/feedback"
)

func TestGitDestinationWritesPerFeedbackFile(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	destination, err := NewGitDestination(repoRoot, repoRoot, config.DestinationConfig{
		Type:      "git",
		Directory: ".feedback",
		Branch:    "feedback",
		Remote:    "origin",
		Push:      boolPtr(false),
	})
	if err != nil {
		t.Fatalf("new git destination: %v", err)
	}

	item := feedback.Item{
		ID:        "97059329-7216-4b46-8cf3-b21223114f3f",
		Provider:  "Claude Code",
		Feedback:  "The CLI hid the real error output.",
		Source:    "cli",
		CreatedAt: time.Date(2026, 5, 6, 1, 59, 30, 0, time.UTC),
	}

	if filepath.IsAbs(destination.directory) {
		t.Fatal("expected relative directory")
	}

	path := filepath.Join(repoRoot, destination.directory, item.ID+".md")
	if rel, err := filepath.Rel(repoRoot, filepath.Join(repoRoot, filepath.Clean(destination.directory))); err != nil || strings.HasPrefix(rel, "..") {
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
	for _, expected := range []string{
		item.ID,
		"The CLI hid the real error output.",
		"_From Claude Code via CLI at 2026-05-06T01:59:30Z_",
	} {
		if !strings.Contains(string(data), expected) {
			t.Fatalf("file missing %q", expected)
		}
	}
}

func TestGitDestinationWritesJSONFileNameAndContent(t *testing.T) {
	t.Parallel()

	repoRoot := t.TempDir()
	destination, err := NewGitDestination(repoRoot, repoRoot, config.DestinationConfig{
		Type:      "git",
		Directory: ".feedback",
		Format:    "json",
		Branch:    "feedback",
		Remote:    "origin",
		Push:      boolPtr(false),
	})
	if err != nil {
		t.Fatalf("new git destination: %v", err)
	}

	item, err := feedback.New("Claude Code", "JSON file output should use a .json extension.", "cli", map[string]any{"kind": "test"})
	if err != nil {
		t.Fatalf("new item: %v", err)
	}

	path := filepath.Join(repoRoot, destination.directory, item.ID+".json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	data, err := item.JSON(true)
	if err != nil {
		t.Fatalf("marshal item: %v", err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	var decoded feedback.Item
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal json file: %v", err)
	}
	if decoded.ID != item.ID || decoded.Provider != item.Provider {
		t.Fatalf("decoded item mismatch: %#v", decoded)
	}
}

func boolPtr(v bool) *bool {
	return &v
}
