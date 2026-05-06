package destinations

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"text/template"
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

// initTestGitRepo creates a bare remote and a working repo with an initial
// commit, suitable for testing the git destination's full Submit flow.
func initTestGitRepo(t *testing.T) (repoRoot, bareRemote string) {
	t.Helper()

	bareRemote = filepath.Join(t.TempDir(), "remote.git")
	repoRoot = t.TempDir()

	for _, args := range [][]string{
		{"init", "--bare", bareRemote},
		{"init", repoRoot},
		{"-C", repoRoot, "config", "user.name", "test"},
		{"-C", repoRoot, "config", "user.email", "test@test"},
		{"-C", repoRoot, "remote", "add", "origin", bareRemote},
	} {
		cmd := exec.CommandContext(context.Background(), "git", args...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %s", args, out)
		}
	}

	// Create an initial commit so HEAD is valid.
	readmePath := filepath.Join(repoRoot, "README.md")
	if err := os.WriteFile(readmePath, []byte("# test\n"), 0o644); err != nil {
		t.Fatalf("write readme: %v", err)
	}
	for _, args := range [][]string{
		{"-C", repoRoot, "add", "."},
		{"-C", repoRoot, "commit", "-m", "initial"},
		{"-C", repoRoot, "push", "origin", "HEAD"},
	} {
		cmd := exec.CommandContext(context.Background(), "git", args...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %s", args, out)
		}
	}

	return repoRoot, bareRemote
}

func TestGitDestinationSubmitMarkdown(t *testing.T) {
	t.Parallel()

	repoRoot, bareRemote := initTestGitRepo(t)

	tmpl, err := template.New("commit-message").Parse("Add feedback entry {{ .ID }}")
	if err != nil {
		t.Fatalf("parse template: %v", err)
	}

	destination := &GitDestination{
		baseDir:       repoRoot,
		repoRoot:      repoRoot,
		branch:        "feedback",
		remote:        "origin",
		directory:     ".feedback",
		format:        "markdown",
		push:          true,
		commitMessage: tmpl,
	}

	item := feedback.Item{
		ID:        "abcdef12-0000-0000-0000-000000000001",
		Provider:  "TestAgent",
		Feedback:  "Full submit flow test",
		Source:    "cli",
		CreatedAt: time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC),
	}

	if err := destination.Submit(context.Background(), item); err != nil {
		t.Fatalf("submit: %v", err)
	}

	// Verify the feedback branch was pushed with the file.
	verifyDir := t.TempDir()
	for _, args := range [][]string{
		{"clone", "--branch", "feedback", bareRemote, verifyDir},
	} {
		cmd := exec.CommandContext(context.Background(), "git", args...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %s", args, out)
		}
	}

	data, err := os.ReadFile(filepath.Join(verifyDir, ".feedback", item.ID+".md"))
	if err != nil {
		t.Fatalf("read pushed file: %v", err)
	}
	if !strings.Contains(string(data), "Full submit flow test") {
		t.Fatalf("pushed file missing feedback text: %s", string(data))
	}
}

func TestGitDestinationSubmitJSON(t *testing.T) {
	t.Parallel()

	repoRoot, bareRemote := initTestGitRepo(t)

	tmpl, err := template.New("commit-message").Parse("Add feedback {{ .ID }}")
	if err != nil {
		t.Fatalf("parse template: %v", err)
	}

	destination := &GitDestination{
		baseDir:       repoRoot,
		repoRoot:      repoRoot,
		branch:        "feedback",
		remote:        "origin",
		directory:     ".feedback",
		format:        "json",
		push:          true,
		commitMessage: tmpl,
	}

	item, err := feedback.New("TestAgent", "JSON submit flow test", "cli", map[string]any{"key": "value"})
	if err != nil {
		t.Fatalf("new item: %v", err)
	}

	if err := destination.Submit(context.Background(), item); err != nil {
		t.Fatalf("submit: %v", err)
	}

	verifyDir := t.TempDir()
	cmd := exec.CommandContext(context.Background(), "git", "clone", "--branch", "feedback", bareRemote, verifyDir)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git clone: %s", out)
	}

	data, err := os.ReadFile(filepath.Join(verifyDir, ".feedback", item.ID+".json"))
	if err != nil {
		t.Fatalf("read pushed file: %v", err)
	}

	var decoded feedback.Item
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal json: %v", err)
	}
	if decoded.ID != item.ID {
		t.Fatalf("unexpected id %q", decoded.ID)
	}
}
