package feedback

import (
	"strings"
	"testing"
	"time"
)

func TestNewRequiresProviderAndFeedback(t *testing.T) {
	t.Parallel()

	if _, err := New("", "feedback", "", nil); err == nil {
		t.Fatal("expected provider validation error")
	}
	if _, err := New("Claude Code", "", "", nil); err == nil {
		t.Fatal("expected feedback validation error")
	}
}

func TestMarkdownEntryIncludesMetadata(t *testing.T) {
	t.Parallel()

	item := Item{
		ID:        "97059329-7216-4b46-8cf3-b21223114f3f",
		Provider:  "Claude Code",
		Feedback:  "Shell errors and confusing stderr output",
		Source:    "cli",
		CreatedAt: time.Date(2026, 5, 6, 1, 59, 30, 0, time.UTC),
		Metadata: map[string]any{
			"command": "git status",
		},
	}

	entry := item.MarkdownEntry()
	for _, expected := range []string{
		"## " + item.ID,
		"Shell errors and confusing stderr output",
		"_From Claude Code via CLI at 2026-05-06T01:59:30Z_",
		"```json",
		"\"command\": \"git status\"",
	} {
		if !strings.Contains(entry, expected) {
			t.Fatalf("entry missing %q", expected)
		}
	}
}
