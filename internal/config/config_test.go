package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAppliesDefaults(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, DefaultFileName)
	if err := os.WriteFile(path, []byte(`{"sinks":[{"type":"file"}]}`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, resolved, err := Load("", dir)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if resolved != path {
		t.Fatalf("expected %s, got %s", path, resolved)
	}
	if cfg.ToolName() != "submit_feedback" {
		t.Fatalf("unexpected tool name %q", cfg.ToolName())
	}
	if cfg.Sinks[0].Path != "FEEDBACK.md" {
		t.Fatalf("unexpected path %q", cfg.Sinks[0].Path)
	}
}

func TestLocateWalksParents(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	nested := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	path := filepath.Join(root, DefaultFileName)
	if err := os.WriteFile(path, []byte(`{"sinks":[{"type":"file"}]}`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	_, resolved, err := Load("", nested)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if resolved != path {
		t.Fatalf("expected %s, got %s", path, resolved)
	}
}
