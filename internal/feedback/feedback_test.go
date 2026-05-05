package feedback

import (
	"strings"
	"testing"
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

	item, err := New("Claude Code", "Shell errors and confusing stderr output", "cli", map[string]any{
		"command": "git status",
	})
	if err != nil {
		t.Fatalf("new item: %v", err)
	}

	entry := item.MarkdownEntry()
	for _, expected := range []string{"## " + item.ID, "Created At:", "Claude Code", "Shell errors and confusing stderr output", "command"} {
		if !strings.Contains(entry, expected) {
			t.Fatalf("entry missing %q", expected)
		}
	}
}
