package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/agorischek/token-for-your-thoughts"
	"github.com/joho/godotenv"
	"github.com/pelletier/go-toml/v2"
	"github.com/xeipuuv/gojsonschema"
)

const (
	DefaultTOMLFileName                 = ".tfyt.toml"
	DefaultJSONFileName                 = ".tfyt.json"
	defaultMCPToolName                  = "submit_feedback"
	defaultMCPDescription               = "Submit feedback about your work context, including tool errors and inefficiencies, as well as information gaps and inconsistencies."
	defaultFeedbackPath                 = "FEEDBACK.md"
	defaultFeedbackJSONPath             = "feedback.jsonl"
	defaultGitRemote                    = "origin"
	defaultGitBranch                    = "feedback"
	defaultGitDirectory                 = ".feedback"
	defaultGitCommitPrefix              = "Add feedback entry"
	defaultCommandMethod                = "submit_feedback"
	defaultCommandContentMode           = "json"
	defaultApplicationInsightsEventName = "tfyt feedback"
	defaultHTTPMethod                   = "POST"
	defaultHTTPTimeoutSeconds           = 10
	defaultGitHubDiscussionsTitle       = "Feedback {{ .ID }} from {{ .Provider }}"
	defaultSQLTable                     = "feedback"
	defaultOTelServiceName              = "tfyt"
)

func defaultConfig() Config {
	return Config{
		Destinations: []DestinationConfig{
			{Type: "file", Path: defaultFeedbackPath, Format: "markdown"},
		},
	}
}

type Config struct {
	Schema       string              `json:"$schema,omitempty" toml:"-"`
	EnvFilePath  string              `json:"env_file_path,omitempty" toml:"env_file_path"`
	MCP          MCPConfig           `json:"mcp,omitempty" toml:"mcp"`
	Destinations []DestinationConfig `json:"destinations,omitempty" toml:"destinations"`
}

type MCPConfig struct {
	ToolName        string `json:"tool_name" toml:"tool_name"`
	ToolDescription string `json:"tool_description" toml:"tool_description"`
}

type DestinationConfig struct {
	Type string `json:"type" toml:"type"`

	Path                string            `json:"path,omitempty" toml:"path"`
	Directory           string            `json:"directory,omitempty" toml:"directory"`
	Format              string            `json:"format,omitempty" toml:"format"`
	URL                 string            `json:"url,omitempty" toml:"url"`
	URLEnv              string            `json:"url_env,omitempty" toml:"url_env"`
	Command             string            `json:"command,omitempty" toml:"command"`
	Args                []string          `json:"args,omitempty" toml:"args"`
	Method              string            `json:"method,omitempty" toml:"method"`
	ContentMode         string            `json:"content_mode,omitempty" toml:"content_mode"`
	TimeoutSeconds      int               `json:"timeout_seconds,omitempty" toml:"timeout_seconds"`
	SuccessStatuses     []int             `json:"success_statuses,omitempty" toml:"success_statuses"`
	ConnectionString    string            `json:"connection_string,omitempty" toml:"connection_string"`
	ConnectionStringEnv string            `json:"connection_string_env,omitempty" toml:"connection_string_env"`
	EventName           string            `json:"event_name,omitempty" toml:"event_name"`
	Repository          string            `json:"repository,omitempty" toml:"repository"`
	Category            string            `json:"category,omitempty" toml:"category"`
	CategoryID          string            `json:"category_id,omitempty" toml:"category_id"`
	Token               string            `json:"token,omitempty" toml:"token"`
	TokenEnv            string            `json:"token_env,omitempty" toml:"token_env"`
	TitleTemplate       string            `json:"title_template,omitempty" toml:"title_template"`
	Branch              string            `json:"branch,omitempty" toml:"branch"`
	Remote              string            `json:"remote,omitempty" toml:"remote"`
	Push                *bool             `json:"push,omitempty" toml:"push"`
	CommitMessage       string            `json:"commit_message,omitempty" toml:"commit_message"`
	Driver              string            `json:"driver,omitempty" toml:"driver"`
	DSN                 string            `json:"dsn,omitempty" toml:"dsn"`
	Table               string            `json:"table,omitempty" toml:"table"`
	AutoCreate          *bool             `json:"auto_create,omitempty" toml:"auto_create"`
	CreateStmt          string            `json:"create_statement,omitempty" toml:"create_statement"`
	InsertStmt          string            `json:"insert_statement,omitempty" toml:"insert_statement"`
	Endpoint            string            `json:"endpoint,omitempty" toml:"endpoint"`
	EndpointEnv         string            `json:"endpoint_env,omitempty" toml:"endpoint_env"`
	Headers             map[string]string `json:"headers,omitempty" toml:"headers"`
	HeadersEnv          string            `json:"headers_env,omitempty" toml:"headers_env"`
	Insecure            bool              `json:"insecure,omitempty" toml:"insecure"`
	ServiceName         string            `json:"service_name,omitempty" toml:"service_name"`
}

func Load(explicitPath, startDir string) (Config, string, error) {
	path, err := locate(explicitPath, startDir)
	if err != nil {
		return Config{}, "", err
	}

	if path == "" {
		cfg := defaultConfig()
		cfg.applyDefaults()
		return cfg, "", nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, "", fmt.Errorf("read config %s: %w", path, err)
	}

	var cfg Config
	if err := decodeConfig(path, data, &cfg); err != nil {
		return Config{}, "", err
	}
	if err := validateAgainstSchema(path, cfg); err != nil {
		return Config{}, "", err
	}

	dotEnv, err := loadDotEnv(filepath.Dir(path), cfg.EnvFilePath)
	if err != nil {
		return Config{}, "", err
	}

	if err := cfg.resolveEnv(dotEnv); err != nil {
		return Config{}, "", err
	}
	cfg.applyDefaults()
	if err := cfg.validate(); err != nil {
		return Config{}, "", err
	}

	return cfg, path, nil
}

func (c Config) ToolName() string {
	return c.MCP.ToolName
}

func (c Config) ToolDescription() string {
	return c.MCP.ToolDescription
}

func (c *Config) applyDefaults() {
	if strings.TrimSpace(c.MCP.ToolName) == "" {
		c.MCP.ToolName = defaultMCPToolName
	}
	if strings.TrimSpace(c.MCP.ToolDescription) == "" {
		c.MCP.ToolDescription = defaultMCPDescription
	}
	if len(c.Destinations) == 0 {
		c.Destinations = []DestinationConfig{{Type: "file"}}
	}

	for i := range c.Destinations {
		destination := &c.Destinations[i]
		destination.Type = strings.ToLower(strings.TrimSpace(destination.Type))

		switch destination.Type {
		case "file":
			destination.Format = normalizeFormat(destination.Format)
			if strings.TrimSpace(destination.Path) == "" {
				if destination.Format == "json" {
					destination.Path = defaultFeedbackJSONPath
				} else {
					destination.Path = defaultFeedbackPath
				}
			}
		case "git":
			destination.Format = normalizeFormat(destination.Format)
			if strings.TrimSpace(destination.Directory) == "" {
				if strings.TrimSpace(destination.Path) != "" {
					destination.Directory = destination.Path
				} else {
					destination.Directory = defaultGitDirectory
				}
			}
			if strings.TrimSpace(destination.Branch) == "" {
				destination.Branch = defaultGitBranch
			}
			if strings.TrimSpace(destination.Remote) == "" {
				destination.Remote = defaultGitRemote
			}
			if destination.Push == nil {
				value := true
				destination.Push = &value
			}
			if strings.TrimSpace(destination.CommitMessage) == "" {
				destination.CommitMessage = defaultGitCommitPrefix + " {{ .ID }}"
			}
		case "command":
			if strings.TrimSpace(destination.ContentMode) == "" {
				destination.ContentMode = defaultCommandContentMode
			}
		case "process":
			if strings.TrimSpace(destination.Method) == "" {
				destination.Method = defaultCommandMethod
			}
		case "http":
			if strings.TrimSpace(destination.Method) == "" {
				destination.Method = defaultHTTPMethod
			}
			if destination.TimeoutSeconds <= 0 {
				destination.TimeoutSeconds = defaultHTTPTimeoutSeconds
			}
		case "application_insights":
			if strings.TrimSpace(destination.EventName) == "" {
				destination.EventName = defaultApplicationInsightsEventName
			}
		case "github_discussions":
			if strings.TrimSpace(destination.TitleTemplate) == "" {
				destination.TitleTemplate = defaultGitHubDiscussionsTitle
			}
		case "sql":
			if strings.TrimSpace(destination.Table) == "" {
				destination.Table = defaultSQLTable
			}
			if destination.AutoCreate == nil {
				value := false
				destination.AutoCreate = &value
			}
		case "otel":
			if strings.TrimSpace(destination.ServiceName) == "" {
				destination.ServiceName = defaultOTelServiceName
			}
		}
	}
}

func (c Config) validate() error {
	for _, destination := range c.Destinations {
		if err := destination.rejectUnknownFields(); err != nil {
			return err
		}

		switch destination.Type {
		case "file":
			if !isSupportedFormat(destination.Format) {
				return fmt.Errorf("file destination format must be markdown or json")
			}
		case "http":
			if strings.TrimSpace(destination.URL) == "" {
				return errors.New("http destination requires url or url_env")
			}
			if strings.TrimSpace(destination.Method) == "" {
				return errors.New("http destination requires method")
			}
			if destination.TimeoutSeconds <= 0 {
				return errors.New("http destination timeout_seconds must be greater than zero")
			}
			for _, status := range destination.SuccessStatuses {
				if status < 100 || status > 599 {
					return fmt.Errorf("http destination success_statuses must contain valid HTTP status codes")
				}
			}
		case "command":
			if strings.TrimSpace(destination.Command) == "" {
				return errors.New("command destination requires command")
			}
			if destination.ContentMode != "json" && destination.ContentMode != "args" {
				return errors.New("command destination content_mode must be json or args")
			}
		case "process":
			if strings.TrimSpace(destination.Command) == "" {
				return errors.New("process destination requires command")
			}
			if strings.TrimSpace(destination.Method) == "" {
				return errors.New("process destination requires method")
			}
		case "application_insights":
			if strings.TrimSpace(destination.ConnectionString) == "" {
				return errors.New("application_insights destination requires connection_string or connection_string_env")
			}
			if strings.TrimSpace(destination.EventName) == "" {
				return errors.New("application_insights destination requires event_name")
			}
		case "github_discussions":
			if strings.TrimSpace(destination.Repository) == "" {
				return errors.New("github_discussions destination requires repository")
			}
			if strings.TrimSpace(destination.Category) == "" && strings.TrimSpace(destination.CategoryID) == "" {
				return errors.New("github_discussions destination requires category or category_id")
			}
			if strings.TrimSpace(destination.Token) == "" {
				return errors.New("github_discussions destination requires token or token_env")
			}
			if strings.TrimSpace(destination.TitleTemplate) == "" {
				return errors.New("github_discussions destination requires title_template")
			}
		case "git":
			if !isSupportedFormat(destination.Format) {
				return fmt.Errorf("git destination format must be markdown or json")
			}
			if strings.TrimSpace(destination.Branch) == "" {
				return errors.New("git destination requires branch")
			}
			if strings.TrimSpace(destination.Remote) == "" {
				return errors.New("git destination requires remote")
			}
			if strings.TrimSpace(destination.Directory) == "" && strings.TrimSpace(destination.Path) == "" {
				return errors.New("git destination requires directory")
			}
		case "sql":
			if strings.TrimSpace(destination.Driver) == "" {
				return errors.New("sql destination requires driver")
			}
			if strings.TrimSpace(destination.DSN) == "" {
				return errors.New("sql destination requires dsn")
			}
			if strings.TrimSpace(destination.InsertStmt) == "" {
				return errors.New("sql destination requires insert_statement")
			}
			if destination.AutoCreate != nil && *destination.AutoCreate && strings.TrimSpace(destination.CreateStmt) == "" {
				return errors.New("sql destination with auto_create requires create_statement")
			}
		case "otel":
		default:
			return fmt.Errorf("unsupported destination type %q", destination.Type)
		}
	}

	return nil
}

var allowedFieldsByDestinationType = map[string]map[string]bool{
	"file": {
		"path":   true,
		"format": true,
	},
	"git": {
		"path":           true,
		"directory":      true,
		"format":         true,
		"branch":         true,
		"remote":         true,
		"push":           true,
		"commit_message": true,
	},
	"command": {
		"command":      true,
		"args":         true,
		"content_mode": true,
	},
	"process": {
		"command": true,
		"args":    true,
		"method":  true,
	},
	"http": {
		"url":              true,
		"url_env":          true,
		"method":           true,
		"headers":          true,
		"headers_env":      true,
		"timeout_seconds":  true,
		"success_statuses": true,
	},
	"application_insights": {
		"connection_string":     true,
		"connection_string_env": true,
		"event_name":            true,
	},
	"github_discussions": {
		"repository":     true,
		"category":       true,
		"category_id":    true,
		"token":          true,
		"token_env":      true,
		"title_template": true,
	},
	"sql": {
		"driver":           true,
		"dsn":              true,
		"table":            true,
		"auto_create":      true,
		"create_statement": true,
		"insert_statement": true,
	},
	"otel": {
		"endpoint":     true,
		"endpoint_env": true,
		"headers":      true,
		"headers_env":  true,
		"insecure":     true,
		"service_name": true,
	},
}

func (s DestinationConfig) rejectUnknownFields() error {
	allowed, ok := allowedFieldsByDestinationType[s.Type]
	if !ok {
		return fmt.Errorf("unsupported destination type %q", s.Type)
	}

	set := func(field string, isSet bool) error {
		if isSet && !allowed[field] {
			return fmt.Errorf("%s destination does not support field %q", s.Type, field)
		}
		return nil
	}

	checks := []struct {
		field string
		isSet bool
	}{
		{"path", strings.TrimSpace(s.Path) != ""},
		{"directory", strings.TrimSpace(s.Directory) != ""},
		{"format", strings.TrimSpace(s.Format) != ""},
		{"url", strings.TrimSpace(s.URL) != ""},
		{"url_env", strings.TrimSpace(s.URLEnv) != ""},
		{"command", strings.TrimSpace(s.Command) != ""},
		{"args", len(s.Args) > 0},
		{"method", strings.TrimSpace(s.Method) != ""},
		{"content_mode", strings.TrimSpace(s.ContentMode) != ""},
		{"timeout_seconds", s.TimeoutSeconds != 0},
		{"success_statuses", len(s.SuccessStatuses) > 0},
		{"connection_string", strings.TrimSpace(s.ConnectionString) != ""},
		{"connection_string_env", strings.TrimSpace(s.ConnectionStringEnv) != ""},
		{"event_name", strings.TrimSpace(s.EventName) != ""},
		{"repository", strings.TrimSpace(s.Repository) != ""},
		{"category", strings.TrimSpace(s.Category) != ""},
		{"category_id", strings.TrimSpace(s.CategoryID) != ""},
		{"token", strings.TrimSpace(s.Token) != ""},
		{"token_env", strings.TrimSpace(s.TokenEnv) != ""},
		{"title_template", strings.TrimSpace(s.TitleTemplate) != ""},
		{"branch", strings.TrimSpace(s.Branch) != ""},
		{"remote", strings.TrimSpace(s.Remote) != ""},
		{"push", s.Push != nil},
		{"commit_message", strings.TrimSpace(s.CommitMessage) != ""},
		{"driver", strings.TrimSpace(s.Driver) != ""},
		{"dsn", strings.TrimSpace(s.DSN) != ""},
		{"table", strings.TrimSpace(s.Table) != ""},
		{"auto_create", s.AutoCreate != nil},
		{"create_statement", strings.TrimSpace(s.CreateStmt) != ""},
		{"insert_statement", strings.TrimSpace(s.InsertStmt) != ""},
		{"endpoint", strings.TrimSpace(s.Endpoint) != ""},
		{"endpoint_env", strings.TrimSpace(s.EndpointEnv) != ""},
		{"headers", len(s.Headers) > 0},
		{"headers_env", strings.TrimSpace(s.HeadersEnv) != ""},
		{"insecure", s.Insecure},
		{"service_name", strings.TrimSpace(s.ServiceName) != ""},
	}

	for _, c := range checks {
		if err := set(c.field, c.isSet); err != nil {
			return err
		}
	}
	return nil
}

func (c *Config) resolveEnv(dotEnv map[string]string) error {
	for i := range c.Destinations {
		destination := &c.Destinations[i]
		switch strings.ToLower(strings.TrimSpace(destination.Type)) {
		case "http":
			value, err := resolveExclusiveEnvString("http url", destination.URL, destination.URLEnv, dotEnv)
			if err != nil {
				return err
			}
			destination.URL = value
			destination.URLEnv = ""

			headers, err := resolveExclusiveEnvMap("http headers", destination.Headers, destination.HeadersEnv, dotEnv)
			if err != nil {
				return err
			}
			destination.Headers = headers
			destination.HeadersEnv = ""
		case "application_insights":
			value, err := resolveExclusiveEnvString("application_insights connection string", destination.ConnectionString, destination.ConnectionStringEnv, dotEnv)
			if err != nil {
				return err
			}
			destination.ConnectionString = value
			destination.ConnectionStringEnv = ""
		case "github_discussions":
			tokenEnv := destination.TokenEnv
			if strings.TrimSpace(destination.Token) == "" && strings.TrimSpace(tokenEnv) == "" {
				tokenEnv = "GITHUB_TOKEN"
			}
			value, err := resolveExclusiveEnvString("github_discussions token", destination.Token, tokenEnv, dotEnv)
			if err != nil {
				return err
			}
			destination.Token = value
			destination.TokenEnv = ""
		case "otel":
			value, err := resolveExclusiveEnvString("otel endpoint", destination.Endpoint, destination.EndpointEnv, dotEnv)
			if err != nil {
				return err
			}
			destination.Endpoint = value
			destination.EndpointEnv = ""

			headers, err := resolveExclusiveEnvMap("otel headers", destination.Headers, destination.HeadersEnv, dotEnv)
			if err != nil {
				return err
			}
			destination.Headers = headers
			destination.HeadersEnv = ""
		}
	}
	return nil
}

func resolveExclusiveEnvString(label, directValue, envName string, dotEnv map[string]string) (string, error) {
	if strings.TrimSpace(envName) == "" {
		return directValue, nil
	}

	value, ok := lookupEnv(strings.TrimSpace(envName), dotEnv)
	if !ok {
		return "", fmt.Errorf("%s env var %q is not set", label, strings.TrimSpace(envName))
	}
	return value, nil
}

func resolveExclusiveEnvMap(label string, directValue map[string]string, envName string, dotEnv map[string]string) (map[string]string, error) {
	if strings.TrimSpace(envName) == "" {
		return directValue, nil
	}

	value, ok := lookupEnv(strings.TrimSpace(envName), dotEnv)
	if !ok {
		return nil, fmt.Errorf("%s env var %q is not set", label, strings.TrimSpace(envName))
	}

	var decoded map[string]string
	if err := json.Unmarshal([]byte(value), &decoded); err != nil {
		return nil, fmt.Errorf("%s env var %q must contain a JSON object: %w", label, strings.TrimSpace(envName), err)
	}
	if decoded == nil {
		decoded = map[string]string{}
	}
	return decoded, nil
}

func normalizeFormat(format string) string {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "", "markdown", "md":
		return "markdown"
	case "json":
		return "json"
	default:
		return strings.ToLower(strings.TrimSpace(format))
	}
}

func isSupportedFormat(format string) bool {
	return format == "markdown" || format == "json"
}

func locate(explicitPath, startDir string) (string, error) {
	if strings.TrimSpace(explicitPath) != "" {
		path := explicitPath
		if !filepath.IsAbs(path) {
			path = filepath.Join(startDir, path)
		}
		return filepath.Abs(path)
	}

	current, err := filepath.Abs(startDir)
	if err != nil {
		return "", err
	}

	for {
		for _, name := range []string{DefaultTOMLFileName, DefaultJSONFileName} {
			candidate := filepath.Join(current, name)
			if _, err := os.Stat(candidate); err == nil {
				return candidate, nil
			}
		}

		parent := filepath.Dir(current)
		if parent == current {
			return "", nil
		}
		current = parent
	}
}

func decodeConfig(path string, data []byte, cfg *Config) error {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".toml":
		decoder := toml.NewDecoder(bytes.NewReader(data))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(cfg); err != nil {
			return fmt.Errorf("decode config %s: %w", path, err)
		}
		return nil
	case ".json":
		decoder := json.NewDecoder(bytes.NewReader(data))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(cfg); err != nil {
			return fmt.Errorf("decode config %s: %w", path, err)
		}
		return nil
	default:
		jsonDecoder := json.NewDecoder(bytes.NewReader(data))
		jsonDecoder.DisallowUnknownFields()
		if err := jsonDecoder.Decode(cfg); err == nil {
			return nil
		}

		tomlDecoder := toml.NewDecoder(bytes.NewReader(data))
		tomlDecoder.DisallowUnknownFields()
		if err := tomlDecoder.Decode(cfg); err == nil {
			return nil
		}
		return fmt.Errorf("decode config %s: unsupported config format", path)
	}
}

func validateAgainstSchema(path string, cfg Config) error {
	schemaLoader := gojsonschema.NewBytesLoader(tfyt.ConfigSchema())
	documentLoader := gojsonschema.NewGoLoader(cfg)

	result, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return fmt.Errorf("validate config schema %s: %w", path, err)
	}
	if result.Valid() {
		return nil
	}

	messages := make([]string, 0, len(result.Errors()))
	for _, issue := range result.Errors() {
		messages = append(messages, issue.String())
	}
	return fmt.Errorf("config %s does not match tfyt.schema.json: %s", path, strings.Join(messages, "; "))
}

func loadDotEnv(baseDir, envFilePath string) (map[string]string, error) {
	if strings.TrimSpace(envFilePath) == "" {
		return nil, nil
	}

	dotEnvPath := envFilePath
	if !filepath.IsAbs(dotEnvPath) {
		dotEnvPath = filepath.Join(baseDir, dotEnvPath)
	}
	if _, err := os.Stat(dotEnvPath); err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("stat env file %s: %w", dotEnvPath, err)
		}
		return nil, fmt.Errorf("stat env file %s: %w", dotEnvPath, err)
	}
	env, err := godotenv.Read(dotEnvPath)
	if err != nil {
		return nil, fmt.Errorf("read env file %s: %w", dotEnvPath, err)
	}
	return env, nil
}

// lookupEnv checks the process environment first, then falls back to the
// dotenv map. This preserves the convention that process env wins over .env.
func lookupEnv(key string, dotEnv map[string]string) (string, bool) {
	if value, ok := os.LookupEnv(key); ok {
		return value, true
	}
	if dotEnv != nil {
		if value, ok := dotEnv[key]; ok {
			return value, true
		}
	}
	return "", false
}
