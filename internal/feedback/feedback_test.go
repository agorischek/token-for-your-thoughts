package feedback

import (
	"strings"
	"testing"
)

func TestNewRequiresProviderAndSummary(t *testing.T) {
	t.Parallel()

	if _, err := New("", "summary", "", "", "", nil); err == nil {
		t.Fatal("expected provider validation error")
	}
	if _, err := New("Claude Code", "", "", "", "", nil); err == nil {
		t.Fatal("expected summary validation error")
	}
}

func TestMarkdownEntryIncludesMetadata(t *testing.T) {
	t.Parallel()

	item, err := New("Claude Code", "Shell errors", "stderr was confusing", "tooling", "cli", map[string]any{
		"command": "git status",
	})
	if err != nil {
		t.Fatalf("new item: %v", err)
	}

	entry := item.MarkdownEntry()
	for _, expected := range []string{"Claude Code", "Shell errors", "stderr was confusing", "command"} {
		if !strings.Contains(entry, expected) {
			t.Fatalf("entry missing %q", expected)
		}
	}
}
