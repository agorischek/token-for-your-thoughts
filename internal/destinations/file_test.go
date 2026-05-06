package destinations

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/agorischek/token-for-your-thoughts/internal/config"
	"github.com/agorischek/token-for-your-thoughts/internal/feedback"
)

func TestFileSinkWritesEntry(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	destination, err := NewFileDestination(dir, config.DestinationConfig{Type: "file", Path: "FEEDBACK.md"})
	if err != nil {
		t.Fatalf("new file destination: %v", err)
	}

	item := feedback.Item{
		ID:        "97059329-7216-4b46-8cf3-b21223114f3f",
		Provider:  "Claude Code",
		Feedback:  "Summary and details together",
		Source:    "cli",
		CreatedAt: time.Date(2026, 5, 6, 1, 59, 30, 0, time.UTC),
	}
	if err := destination.Submit(context.Background(), item); err != nil {
		t.Fatalf("submit: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "FEEDBACK.md"))
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	expected := "# Feedback\n\nRuntime feedback entries collected by `tfyt` will appear here when the file destination is enabled.\n\n## 97059329-7216-4b46-8cf3-b21223114f3f\n\nSummary and details together\n\n_From Claude Code via CLI at 2026-05-06T01:59:30Z_\n"
	if string(data) != expected {
		t.Fatalf("unexpected markdown output:\n%s", string(data))
	}
}

func TestFileDestinationPreservesSingleBlankLineBetweenEntries(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	destination, err := NewFileDestination(dir, config.DestinationConfig{Type: "file", Path: "FEEDBACK.md"})
	if err != nil {
		t.Fatalf("new file destination: %v", err)
	}

	first := feedback.Item{
		ID:        "97059329-7216-4b46-8cf3-b21223114f3f",
		Provider:  "Codex",
		Feedback:  "config validation smoke",
		Source:    "cli",
		CreatedAt: time.Date(2026, 5, 6, 1, 59, 30, 0, time.UTC),
	}
	second := feedback.Item{
		ID:        "d0d08407-c381-43ab-bd9e-a74f3c0224ef",
		Provider:  "GitHub Copilot CLI",
		Feedback:  "Test feedback from Copilot CLI via MCP. Do not use shell commands or edit files.",
		Source:    "mcp",
		CreatedAt: time.Date(2026, 5, 6, 2, 1, 59, 0, time.UTC),
		Metadata: map[string]any{
			"source": "mcp",
		},
	}

	if err := destination.Submit(context.Background(), first); err != nil {
		t.Fatalf("submit first: %v", err)
	}
	if err := destination.Submit(context.Background(), second); err != nil {
		t.Fatalf("submit second: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "FEEDBACK.md"))
	if err != nil {
		t.Fatalf("read file: %v", err)
	}

	expected := "# Feedback\n\nRuntime feedback entries collected by `tfyt` will appear here when the file destination is enabled.\n\n## 97059329-7216-4b46-8cf3-b21223114f3f\n\nconfig validation smoke\n\n_From Codex via CLI at 2026-05-06T01:59:30Z_\n\n## d0d08407-c381-43ab-bd9e-a74f3c0224ef\n\nTest feedback from Copilot CLI via MCP. Do not use shell commands or edit files.\n\n_From GitHub Copilot CLI via MCP at 2026-05-06T02:01:59Z_\n\n```json\n{\n  \"source\": \"mcp\"\n}\n```\n"
	if string(data) != expected {
		t.Fatalf("unexpected markdown output:\n%s", string(data))
	}
}

func TestFileSinkWritesJSONEntry(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	destination, err := NewFileDestination(dir, config.DestinationConfig{Type: "file", Path: "FEEDBACK.jsonl", Format: "json"})
	if err != nil {
		t.Fatalf("new file destination: %v", err)
	}

	item, err := feedback.New("Claude Code", "JSON output should be machine-friendly.", "cli", map[string]any{"key": "value"})
	if err != nil {
		t.Fatalf("new item: %v", err)
	}
	if err := destination.Submit(context.Background(), item); err != nil {
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
