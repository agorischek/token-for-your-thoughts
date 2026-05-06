package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAppliesDefaults(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, DefaultJSONFileName)
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

func TestLoadAcceptsJSONSchemaProperty(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, DefaultJSONFileName)
	if err := os.WriteFile(path, []byte(`{"$schema":"./tfyt.schema.json","sinks":[{"type":"file"}]}`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, _, err := Load("", dir)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.Schema != "./tfyt.schema.json" {
		t.Fatalf("unexpected schema %q", cfg.Schema)
	}
}

func TestLoadAcceptsEnvFilePath(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, DefaultJSONFileName)
	if err := os.WriteFile(path, []byte(`{"env_file_path":".env","sinks":[{"type":"file"}]}`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("TEST=1\n"), 0o644); err != nil {
		t.Fatalf("write env file: %v", err)
	}

	cfg, _, err := Load("", dir)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.EnvFilePath != ".env" {
		t.Fatalf("unexpected env file path %q", cfg.EnvFilePath)
	}
}

func TestLoadParsesTOML(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, DefaultTOMLFileName)
	if err := os.WriteFile(path, []byte(`
[mcp]
tool_name = "submit_feedback"

[[sinks]]
type = "file"
format = "json"
`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, resolved, err := Load("", dir)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if resolved != path {
		t.Fatalf("expected %s, got %s", path, resolved)
	}
	if cfg.Sinks[0].Path != "feedback.jsonl" {
		t.Fatalf("unexpected path %q", cfg.Sinks[0].Path)
	}
	if cfg.Sinks[0].Format != "json" {
		t.Fatalf("unexpected format %q", cfg.Sinks[0].Format)
	}
}

func TestLoadDefaultsToFileSinkWhenSinksMissing(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, DefaultJSONFileName)
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
	path := filepath.Join(dir, DefaultJSONFileName)
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
	path := filepath.Join(dir, DefaultJSONFileName)
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
	path := filepath.Join(dir, DefaultJSONFileName)
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

func TestLoadAppliesHTTPDefaults(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, DefaultJSONFileName)
	if err := os.WriteFile(path, []byte(`{"sinks":[{"type":"http","url":"https://example.com/feedback"}]}`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, _, err := Load("", dir)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.Sinks[0].Method != "POST" {
		t.Fatalf("unexpected method %q", cfg.Sinks[0].Method)
	}
	if cfg.Sinks[0].TimeoutSeconds != 10 {
		t.Fatalf("unexpected timeout %d", cfg.Sinks[0].TimeoutSeconds)
	}
}

func TestLoadAppliesApplicationInsightsDefaults(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, DefaultJSONFileName)
	if err := os.WriteFile(path, []byte(`{"sinks":[{"type":"application_insights","connection_string":"InstrumentationKey=abc"}]}`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, _, err := Load("", dir)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.Sinks[0].EventName != "tfyt feedback" {
		t.Fatalf("unexpected event name %q", cfg.Sinks[0].EventName)
	}
}

func TestLoadResolvesApplicationInsightsConnectionStringEnv(t *testing.T) {
	t.Setenv("APPINSIGHTS_CONNECTION_STRING", "InstrumentationKey=abc;IngestionEndpoint=https://example.com/")

	dir := t.TempDir()
	path := filepath.Join(dir, DefaultJSONFileName)
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
	path := filepath.Join(dir, DefaultJSONFileName)
	if err := os.WriteFile(path, []byte(`{"sinks":[{"type":"application_insights","connection_string":"InstrumentationKey=def","connection_string_env":"APPINSIGHTS_CONNECTION_STRING"}]}`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if _, _, err := Load("", dir); err == nil {
		t.Fatal("expected config load error")
	}
}

func TestLoadRejectsMissingApplicationInsightsEnv(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, DefaultJSONFileName)
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
	path := filepath.Join(dir, DefaultJSONFileName)
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

func TestLoadResolvesHTTPEnvFields(t *testing.T) {
	t.Setenv("TFYT_HTTP_URL", "https://example.com/feedback")
	t.Setenv("TFYT_HTTP_HEADERS", `{"Authorization":"Bearer test-token"}`)

	dir := t.TempDir()
	path := filepath.Join(dir, DefaultJSONFileName)
	if err := os.WriteFile(path, []byte(`{"sinks":[{"type":"http","url_env":"TFYT_HTTP_URL","headers_env":"TFYT_HTTP_HEADERS"}]}`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, _, err := Load("", dir)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.Sinks[0].URL != "https://example.com/feedback" {
		t.Fatalf("unexpected url %q", cfg.Sinks[0].URL)
	}
	if cfg.Sinks[0].Headers["Authorization"] != "Bearer test-token" {
		t.Fatalf("unexpected authorization header %q", cfg.Sinks[0].Headers["Authorization"])
	}
}

func TestLoadRejectsBothHTTPURLSources(t *testing.T) {
	t.Setenv("TFYT_HTTP_URL", "https://example.com/feedback")

	dir := t.TempDir()
	path := filepath.Join(dir, DefaultJSONFileName)
	if err := os.WriteFile(path, []byte(`{"sinks":[{"type":"http","url":"https://direct.example/feedback","url_env":"TFYT_HTTP_URL"}]}`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if _, _, err := Load("", dir); err == nil {
		t.Fatal("expected config load error")
	}
}

func TestLoadRejectsBothHTTPHeaderSources(t *testing.T) {
	t.Setenv("TFYT_HTTP_HEADERS", `{"Authorization":"Bearer test-token"}`)

	dir := t.TempDir()
	path := filepath.Join(dir, DefaultJSONFileName)
	if err := os.WriteFile(path, []byte(`{"sinks":[{"type":"http","url":"https://example.com/feedback","headers":{"Authorization":"Bearer direct-token"},"headers_env":"TFYT_HTTP_HEADERS"}]}`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if _, _, err := Load("", dir); err == nil {
		t.Fatal("expected config load error")
	}
}

func TestLoadRejectsBothOTelEndpointSources(t *testing.T) {
	t.Setenv("OTEL_ENDPOINT", "https://example.com/v1/logs")

	dir := t.TempDir()
	path := filepath.Join(dir, DefaultJSONFileName)
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
	path := filepath.Join(dir, DefaultJSONFileName)
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
	path := filepath.Join(dir, DefaultJSONFileName)
	if err := os.WriteFile(path, []byte(`{"sinks":[{"type":"otel","headers_env":"OTEL_HEADERS"}]}`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if _, _, err := Load("", dir); err == nil {
		t.Fatal("expected config load error")
	}
}

func TestLoadAutoloadsDotEnv(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, DefaultJSONFileName)
	if err := os.WriteFile(configPath, []byte(`{"env_file_path":".env","sinks":[{"type":"otel","endpoint_env":"TEST_SUGGESTING_OTEL_ENDPOINT","headers_env":"TEST_SUGGESTING_OTEL_HEADERS"}]}`), 0o644); err != nil {
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
	configPath := filepath.Join(dir, DefaultJSONFileName)
	if err := os.WriteFile(configPath, []byte(`{"env_file_path":".env","sinks":[{"type":"otel","endpoint_env":"BETTER_STACK_OTEL_ENDPOINT"}]}`), 0o644); err != nil {
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

func TestLoadDoesNotAutoloadDotEnvWithoutEnvFilePath(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, DefaultJSONFileName)
	if err := os.WriteFile(configPath, []byte(`{"sinks":[{"type":"otel","endpoint_env":"BETTER_STACK_OTEL_ENDPOINT"}]}`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	dotEnvPath := filepath.Join(dir, ".env")
	if err := os.WriteFile(dotEnvPath, []byte("BETTER_STACK_OTEL_ENDPOINT=https://from-dotenv.example/v1/logs\n"), 0o644); err != nil {
		t.Fatalf("write .env: %v", err)
	}

	if _, _, err := Load("", dir); err == nil {
		t.Fatal("expected config load error")
	}
}

func TestLoadResolvesRelativeEnvFilePathFromConfigDirectory(t *testing.T) {
	dir := t.TempDir()
	configDir := filepath.Join(dir, "config")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	configPath := filepath.Join(configDir, DefaultJSONFileName)
	if err := os.WriteFile(configPath, []byte(`{"env_file_path":"../shared.env","sinks":[{"type":"otel","endpoint_env":"BETTER_STACK_OTEL_ENDPOINT"}]}`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	dotEnvPath := filepath.Join(dir, "shared.env")
	if err := os.WriteFile(dotEnvPath, []byte("BETTER_STACK_OTEL_ENDPOINT=https://from-relative-env.example/v1/logs\n"), 0o644); err != nil {
		t.Fatalf("write env file: %v", err)
	}

	cfg, _, err := Load("", configDir)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.Sinks[0].Endpoint != "https://from-relative-env.example/v1/logs" {
		t.Fatalf("unexpected endpoint %q", cfg.Sinks[0].Endpoint)
	}
}

func TestLoadAppliesJSONFileDefaults(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, DefaultJSONFileName)
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

	path := filepath.Join(root, DefaultJSONFileName)
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

func TestLoadPrefersTOMLOverJSON(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	tomlPath := filepath.Join(dir, DefaultTOMLFileName)
	if err := os.WriteFile(tomlPath, []byte(`
[[sinks]]
type = "file"
format = "json"
`), 0o644); err != nil {
		t.Fatalf("write toml config: %v", err)
	}

	jsonPath := filepath.Join(dir, DefaultJSONFileName)
	if err := os.WriteFile(jsonPath, []byte(`{"sinks":[{"type":"file","format":"markdown"}]}`), 0o644); err != nil {
		t.Fatalf("write json config: %v", err)
	}

	cfg, resolved, err := Load("", dir)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if resolved != tomlPath {
		t.Fatalf("expected %s, got %s", tomlPath, resolved)
	}
	if cfg.Sinks[0].Format != "json" {
		t.Fatalf("expected toml config to win, got format %q", cfg.Sinks[0].Format)
	}
}

func TestLoadValidatesAgainstSchema(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, DefaultJSONFileName)
	if err := os.WriteFile(path, []byte(`{"sinks":[{"type":"http"}]}`), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if _, _, err := Load("", dir); err == nil {
		t.Fatal("expected schema validation error")
	}
}
