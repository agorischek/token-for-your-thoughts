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
	if cfg.Sinks[0].Format != "markdown" {
		t.Fatalf("unexpected format %q", cfg.Sinks[0].Format)
	}
}

func TestLoadDefaultsToFileSinkWhenSinksMissing(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, DefaultFileName)
	if err := os.WriteFile(path, []byte(`{}`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, _, err := Load("", dir)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if len(cfg.Sinks) != 1 {
		t.Fatalf("expected 1 sink, got %d", len(cfg.Sinks))
	}
	if cfg.Sinks[0].Type != "file" {
		t.Fatalf("unexpected sink type %q", cfg.Sinks[0].Type)
	}
	if cfg.Sinks[0].Path != "FEEDBACK.md" {
		t.Fatalf("unexpected path %q", cfg.Sinks[0].Path)
	}
}

func TestLoadDefaultsToFileSinkWhenSinksEmpty(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, DefaultFileName)
	if err := os.WriteFile(path, []byte(`{"sinks":[]}`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, _, err := Load("", dir)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if len(cfg.Sinks) != 1 {
		t.Fatalf("expected 1 sink, got %d", len(cfg.Sinks))
	}
	if cfg.Sinks[0].Type != "file" {
		t.Fatalf("unexpected sink type %q", cfg.Sinks[0].Type)
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
	if cfg.Sinks[0].Format != "markdown" {
		t.Fatalf("unexpected format %q", cfg.Sinks[0].Format)
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

func TestLoadResolvesApplicationInsightsConnectionStringEnv(t *testing.T) {
	t.Setenv("APPINSIGHTS_CONNECTION_STRING", "InstrumentationKey=abc;IngestionEndpoint=https://example.com/")

	dir := t.TempDir()
	path := filepath.Join(dir, DefaultFileName)
	if err := os.WriteFile(path, []byte(`{"sinks":[{"type":"application_insights","connection_string_env":"APPINSIGHTS_CONNECTION_STRING"}]}`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, _, err := Load("", dir)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.Sinks[0].ConnectionString != "InstrumentationKey=abc;IngestionEndpoint=https://example.com/" {
		t.Fatalf("unexpected connection string %q", cfg.Sinks[0].ConnectionString)
	}
}

func TestLoadRejectsBothApplicationInsightsConnectionStringSources(t *testing.T) {
	t.Setenv("APPINSIGHTS_CONNECTION_STRING", "InstrumentationKey=abc")

	dir := t.TempDir()
	path := filepath.Join(dir, DefaultFileName)
	if err := os.WriteFile(path, []byte(`{"sinks":[{"type":"application_insights","connection_string":"InstrumentationKey=def","connection_string_env":"APPINSIGHTS_CONNECTION_STRING"}]}`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if _, _, err := Load("", dir); err == nil {
		t.Fatal("expected config load error")
	}
}

func TestLoadRejectsMissingApplicationInsightsEnv(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, DefaultFileName)
	if err := os.WriteFile(path, []byte(`{"sinks":[{"type":"application_insights","connection_string_env":"APPINSIGHTS_CONNECTION_STRING"}]}`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if _, _, err := Load("", dir); err == nil {
		t.Fatal("expected config load error")
	}
}

func TestLoadResolvesOTelEnvFields(t *testing.T) {
	t.Setenv("OTEL_ENDPOINT", "https://example.com/v1/logs")
	t.Setenv("OTEL_HEADERS", `{"Authorization":"Bearer test-token"}`)

	dir := t.TempDir()
	path := filepath.Join(dir, DefaultFileName)
	if err := os.WriteFile(path, []byte(`{"sinks":[{"type":"otel","endpoint_env":"OTEL_ENDPOINT","headers_env":"OTEL_HEADERS"}]}`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, _, err := Load("", dir)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.Sinks[0].Endpoint != "https://example.com/v1/logs" {
		t.Fatalf("unexpected endpoint %q", cfg.Sinks[0].Endpoint)
	}
	if cfg.Sinks[0].Headers["Authorization"] != "Bearer test-token" {
		t.Fatalf("unexpected authorization header %q", cfg.Sinks[0].Headers["Authorization"])
	}
}

func TestLoadRejectsBothOTelEndpointSources(t *testing.T) {
	t.Setenv("OTEL_ENDPOINT", "https://example.com/v1/logs")

	dir := t.TempDir()
	path := filepath.Join(dir, DefaultFileName)
	if err := os.WriteFile(path, []byte(`{"sinks":[{"type":"otel","endpoint":"https://direct.example/v1/logs","endpoint_env":"OTEL_ENDPOINT"}]}`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if _, _, err := Load("", dir); err == nil {
		t.Fatal("expected config load error")
	}
}

func TestLoadRejectsBothOTelHeadersSources(t *testing.T) {
	t.Setenv("OTEL_HEADERS", `{"Authorization":"Bearer test-token"}`)

	dir := t.TempDir()
	path := filepath.Join(dir, DefaultFileName)
	if err := os.WriteFile(path, []byte(`{"sinks":[{"type":"otel","headers":{"Authorization":"Bearer direct-token"},"headers_env":"OTEL_HEADERS"}]}`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if _, _, err := Load("", dir); err == nil {
		t.Fatal("expected config load error")
	}
}

func TestLoadRejectsInvalidOTelHeadersEnv(t *testing.T) {
	t.Setenv("OTEL_HEADERS", `not-json`)

	dir := t.TempDir()
	path := filepath.Join(dir, DefaultFileName)
	if err := os.WriteFile(path, []byte(`{"sinks":[{"type":"otel","headers_env":"OTEL_HEADERS"}]}`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if _, _, err := Load("", dir); err == nil {
		t.Fatal("expected config load error")
	}
}

func TestLoadAutoloadsDotEnv(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, DefaultFileName)
	if err := os.WriteFile(configPath, []byte(`{"sinks":[{"type":"otel","endpoint_env":"TEST_SUGGESTING_OTEL_ENDPOINT","headers_env":"TEST_SUGGESTING_OTEL_HEADERS"}]}`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	dotEnvPath := filepath.Join(dir, ".env")
	if err := os.WriteFile(dotEnvPath, []byte("TEST_SUGGESTING_OTEL_ENDPOINT=https://example.com/v1/logs\nTEST_SUGGESTING_OTEL_HEADERS='{\"Authorization\":\"Bearer from-dotenv\"}'\n"), 0o644); err != nil {
		t.Fatalf("write .env: %v", err)
	}

	cfg, _, err := Load("", dir)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.Sinks[0].Endpoint != "https://example.com/v1/logs" {
		t.Fatalf("unexpected endpoint %q", cfg.Sinks[0].Endpoint)
	}
	if cfg.Sinks[0].Headers["Authorization"] != "Bearer from-dotenv" {
		t.Fatalf("unexpected authorization header %q", cfg.Sinks[0].Headers["Authorization"])
	}
}

func TestLoadDotEnvDoesNotOverrideExistingEnv(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, DefaultFileName)
	if err := os.WriteFile(configPath, []byte(`{"sinks":[{"type":"otel","endpoint_env":"BETTER_STACK_OTEL_ENDPOINT"}]}`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	dotEnvPath := filepath.Join(dir, ".env")
	if err := os.WriteFile(dotEnvPath, []byte("BETTER_STACK_OTEL_ENDPOINT=https://from-dotenv.example/v1/logs\n"), 0o644); err != nil {
		t.Fatalf("write .env: %v", err)
	}

	t.Setenv("BETTER_STACK_OTEL_ENDPOINT", "https://from-process.example/v1/logs")

	cfg, _, err := Load("", dir)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.Sinks[0].Endpoint != "https://from-process.example/v1/logs" {
		t.Fatalf("unexpected endpoint %q", cfg.Sinks[0].Endpoint)
	}
}

func TestLoadAppliesJSONFileDefaults(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, DefaultFileName)
	if err := os.WriteFile(path, []byte(`{"sinks":[{"type":"file","format":"json"}]}`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, _, err := Load("", dir)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.Sinks[0].Path != "feedback.jsonl" {
		t.Fatalf("unexpected path %q", cfg.Sinks[0].Path)
	}
	if cfg.Sinks[0].Format != "json" {
		t.Fatalf("unexpected format %q", cfg.Sinks[0].Format)
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
