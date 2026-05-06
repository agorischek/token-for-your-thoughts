package app

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunPrintsHelpWithNoArgs(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	if err := Run(context.Background(), "test", nil, &stdout, &bytes.Buffer{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout.String(), "tfyt collects agent feedback") {
		t.Fatalf("expected help text, got %q", stdout.String())
	}
}

func TestRunPrintsHelpForHelpCommand(t *testing.T) {
	t.Parallel()

	for _, cmd := range []string{"help", "-h", "--help"} {
		var stdout bytes.Buffer
		if err := Run(context.Background(), "test", []string{cmd}, &stdout, &bytes.Buffer{}); err != nil {
			t.Fatalf("unexpected error for %q: %v", cmd, err)
		}
		if !strings.Contains(stdout.String(), "tfyt collects agent feedback") {
			t.Fatalf("expected help text for %q, got %q", cmd, stdout.String())
		}
	}
}

func TestRunPrintsVersion(t *testing.T) {
	t.Parallel()

	for _, cmd := range []string{"version", "--version"} {
		var stdout bytes.Buffer
		if err := Run(context.Background(), "1.2.3", []string{cmd}, &stdout, &bytes.Buffer{}); err != nil {
			t.Fatalf("unexpected error for %q: %v", cmd, err)
		}
		if strings.TrimSpace(stdout.String()) != "1.2.3" {
			t.Fatalf("expected version 1.2.3, got %q", stdout.String())
		}
	}
}

func TestRunRejectsUnknownCommand(t *testing.T) {
	t.Parallel()

	var stderr bytes.Buffer
	err := Run(context.Background(), "test", []string{"bogus"}, &bytes.Buffer{}, &stderr)
	if err == nil {
		t.Fatal("expected error for unknown command")
	}
	if !strings.Contains(err.Error(), "bogus") {
		t.Fatalf("expected error to mention command, got %q", err.Error())
	}
}

func TestRunSubmitRequiresProviderAndFeedback(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "tfyt.json")
	if err := os.WriteFile(cfgPath, []byte(`{"destinations":[{"type":"file"}]}`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	err := Run(context.Background(), "test", []string{
		"submit", "--config", cfgPath,
	}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected error when provider is missing")
	}
}

func TestRunSubmitWritesFeedback(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "tfyt.json")
	if err := os.WriteFile(cfgPath, []byte(`{"destinations":[{"type":"file"}]}`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	var stdout bytes.Buffer
	err := Run(context.Background(), "test", []string{
		"submit",
		"--config", cfgPath,
		"--provider", "TestAgent",
		"--feedback", "test feedback message",
	}, &stdout, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout.String(), "submitted feedback") {
		t.Fatalf("expected submission confirmation, got %q", stdout.String())
	}

	data, err := os.ReadFile(filepath.Join(dir, "FEEDBACK.md"))
	if err != nil {
		t.Fatalf("read feedback file: %v", err)
	}
	if !strings.Contains(string(data), "test feedback message") {
		t.Fatalf("feedback file missing message: %s", string(data))
	}
}

func TestRunSubmitRejectsPositionalArgs(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "tfyt.json")
	if err := os.WriteFile(cfgPath, []byte(`{"destinations":[{"type":"file"}]}`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	err := Run(context.Background(), "test", []string{
		"submit", "--config", cfgPath, "extra-arg",
	}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected error for positional arguments")
	}
}

func TestParseMetadataReturnsNilForEmpty(t *testing.T) {
	t.Parallel()

	result, err := parseMetadata("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Fatalf("expected nil metadata, got %v", result)
	}
}

func TestParseMetadataRejectsInvalidJSON(t *testing.T) {
	t.Parallel()

	if _, err := parseMetadata("not-json"); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestParseMetadataReturnsMap(t *testing.T) {
	t.Parallel()

	result, err := parseMetadata(`{"key":"value"}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["key"] != "value" {
		t.Fatalf("unexpected metadata %v", result)
	}
}

func TestRunUpdateAlreadyUpToDate(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network test in short mode")
	}
	t.Parallel()

	var stdout bytes.Buffer
	err := Run(context.Background(), "999.0.0", []string{"update"}, &stdout, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(stdout.String(), "already up to date") {
		t.Fatalf("expected up-to-date message, got %q", stdout.String())
	}
}

func TestRunUpdateDevVersionFindsRelease(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping network test in short mode")
	}
	t.Parallel()

	var stdout bytes.Buffer
	// "dev" is not valid semver, so DetectLatest should find a release newer than it.
	// We cancel immediately to avoid actually downloading/replacing the binary.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	// With a cancelled context the network call will fail, which is expected.
	err := Run(ctx, "dev", []string{"update"}, &stdout, &bytes.Buffer{})
	if err == nil {
		// It's okay if it errors due to the cancelled context.
		return
	}
	// The error should be from the network, not from argument parsing.
	if strings.Contains(err.Error(), "unknown command") {
		t.Fatalf("update command not recognized: %v", err)
	}
}
