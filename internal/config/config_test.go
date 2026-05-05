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

func TestLoadAppliesGitDefaults(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, DefaultFileName)
	if err := os.WriteFile(path, []byte(`{"sinks":[{"type":"git"}]}`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, _, err := Load("", dir)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.Sinks[0].Branch != "feedback" {
		t.Fatalf("unexpected branch %q", cfg.Sinks[0].Branch)
	}
	if cfg.Sinks[0].Directory != ".feedback" {
		t.Fatalf("unexpected directory %q", cfg.Sinks[0].Directory)
	}
}

func TestLoadAppliesCommandDefaults(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, DefaultFileName)
	if err := os.WriteFile(path, []byte(`{"sinks":[{"type":"command","command":"bridge"}]}`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, _, err := Load("", dir)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.Sinks[0].Method != "submit_feedback" {
		t.Fatalf("unexpected method %q", cfg.Sinks[0].Method)
	}
}

func TestLoadAppliesApplicationInsightsDefaults(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, DefaultFileName)
	if err := os.WriteFile(path, []byte(`{"sinks":[{"type":"application_insights","connection_string":"InstrumentationKey=abc"}]}`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, _, err := Load("", dir)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.Sinks[0].EventName != "suggesting feedback" {
		t.Fatalf("unexpected event name %q", cfg.Sinks[0].EventName)
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
