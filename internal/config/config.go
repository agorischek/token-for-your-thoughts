package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
)

const (
	DefaultFileName                     = ".tfyt.json"
	defaultMCPToolName                  = "submit_feedback"
	defaultMCPDescription               = "Submit feedback about your work context, including tool errors and inefficiencies, as well as information gaps and inconsistencies."
	defaultFeedbackPath                 = "FEEDBACK.md"
	defaultFeedbackJSONPath             = "feedback.jsonl"
	defaultGitRemote                    = "origin"
	defaultGitBranch                    = "feedback"
	defaultGitDirectory                 = ".feedback"
	defaultGitCommitPrefix              = "Add feedback entry"
	defaultCommandMethod                = "submit_feedback"
	defaultApplicationInsightsEventName = "tfyt feedback"
	defaultSQLTable                     = "feedback"
	defaultOTelServiceName              = "tfyt"
)

type Config struct {
	MCP   MCPConfig    `json:"mcp"`
	Sinks []SinkConfig `json:"sinks"`
}

type MCPConfig struct {
	ToolName        string `json:"tool_name"`
	ToolDescription string `json:"tool_description"`
}

type SinkConfig struct {
	Type string `json:"type"`

	Path                string            `json:"path,omitempty"`
	Directory           string            `json:"directory,omitempty"`
	Format              string            `json:"format,omitempty"`
	Command             string            `json:"command,omitempty"`
	Args                []string          `json:"args,omitempty"`
	Method              string            `json:"method,omitempty"`
	ConnectionString    string            `json:"connection_string,omitempty"`
	ConnectionStringEnv string            `json:"connection_string_env,omitempty"`
	EventName           string            `json:"event_name,omitempty"`
	Branch              string            `json:"branch,omitempty"`
	Remote              string            `json:"remote,omitempty"`
	Push                *bool             `json:"push,omitempty"`
	CommitMessage       string            `json:"commit_message,omitempty"`
	Driver              string            `json:"driver,omitempty"`
	DSN                 string            `json:"dsn,omitempty"`
	Table               string            `json:"table,omitempty"`
	AutoCreate          *bool             `json:"auto_create,omitempty"`
	CreateStmt          string            `json:"create_statement,omitempty"`
	InsertStmt          string            `json:"insert_statement,omitempty"`
	Endpoint            string            `json:"endpoint,omitempty"`
	EndpointEnv         string            `json:"endpoint_env,omitempty"`
	Headers             map[string]string `json:"headers,omitempty"`
	HeadersEnv          string            `json:"headers_env,omitempty"`
	Insecure            bool              `json:"insecure,omitempty"`
	ServiceName         string            `json:"service_name,omitempty"`
}

func Load(explicitPath, startDir string) (Config, string, error) {
	path, err := locate(explicitPath, startDir)
	if err != nil {
		return Config{}, "", err
	}

	dotEnv, err := loadDotEnv(filepath.Dir(path))
	if err != nil {
		return Config{}, "", err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, "", fmt.Errorf("read config %s: %w", path, err)
	}

	var cfg Config
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&cfg); err != nil {
		return Config{}, "", fmt.Errorf("decode config %s: %w", path, err)
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
	if strings.TrimSpace(c.MCP.ToolName) != "" {
		return c.MCP.ToolName
	}
	return defaultMCPToolName
}

func (c Config) ToolDescription() string {
	if strings.TrimSpace(c.MCP.ToolDescription) != "" {
		return c.MCP.ToolDescription
	}
	return defaultMCPDescription
}

func (c *Config) applyDefaults() {
	if strings.TrimSpace(c.MCP.ToolName) == "" {
		c.MCP.ToolName = defaultMCPToolName
	}
	if strings.TrimSpace(c.MCP.ToolDescription) == "" {
		c.MCP.ToolDescription = defaultMCPDescription
	}
	if len(c.Sinks) == 0 {
		c.Sinks = []SinkConfig{{Type: "file"}}
	}

	for i := range c.Sinks {
		sink := &c.Sinks[i]
		sink.Type = strings.ToLower(strings.TrimSpace(sink.Type))
		sink.Format = normalizeFormat(sink.Format)

		switch sink.Type {
		case "file":
			if strings.TrimSpace(sink.Path) == "" {
				if sink.Format == "json" {
					sink.Path = defaultFeedbackJSONPath
				} else {
					sink.Path = defaultFeedbackPath
				}
			}
		case "git":
			if strings.TrimSpace(sink.Directory) == "" {
				if strings.TrimSpace(sink.Path) != "" {
					sink.Directory = sink.Path
				} else {
					sink.Directory = defaultGitDirectory
				}
			}
			if strings.TrimSpace(sink.Branch) == "" {
				sink.Branch = defaultGitBranch
			}
			if strings.TrimSpace(sink.Remote) == "" {
				sink.Remote = defaultGitRemote
			}
			if sink.Push == nil {
				value := true
				sink.Push = &value
			}
			if strings.TrimSpace(sink.CommitMessage) == "" {
				sink.CommitMessage = defaultGitCommitPrefix + " {{ .ID }}"
			}
		case "command":
			if strings.TrimSpace(sink.Method) == "" {
				sink.Method = defaultCommandMethod
			}
		case "application_insights":
			if strings.TrimSpace(sink.EventName) == "" {
				sink.EventName = defaultApplicationInsightsEventName
			}
		case "sql":
			if strings.TrimSpace(sink.Table) == "" {
				sink.Table = defaultSQLTable
			}
			if sink.AutoCreate == nil {
				value := false
				sink.AutoCreate = &value
			}
		case "otel":
			if strings.TrimSpace(sink.ServiceName) == "" {
				sink.ServiceName = defaultOTelServiceName
			}
		}
	}
}

func (c Config) validate() error {
	for _, sink := range c.Sinks {
		if err := sink.rejectUnknownFields(); err != nil {
			return err
		}

		switch sink.Type {
		case "file":
			if !isSupportedFormat(sink.Format) {
				return fmt.Errorf("file sink format must be markdown or json")
			}
		case "command":
			if strings.TrimSpace(sink.Command) == "" {
				return errors.New("command sink requires command")
			}
			if strings.TrimSpace(sink.Method) == "" {
				return errors.New("command sink requires method")
			}
		case "application_insights":
			if hasBothValues(sink.ConnectionString, sink.ConnectionStringEnv) {
				return errors.New("application_insights sink cannot set both connection_string and connection_string_env")
			}
			if strings.TrimSpace(sink.ConnectionString) == "" {
				return errors.New("application_insights sink requires connection_string or connection_string_env")
			}
			if strings.TrimSpace(sink.EventName) == "" {
				return errors.New("application_insights sink requires event_name")
			}
		case "git":
			if !isSupportedFormat(sink.Format) {
				return fmt.Errorf("git sink format must be markdown or json")
			}
			if strings.TrimSpace(sink.Branch) == "" {
				return errors.New("git sink requires branch")
			}
			if strings.TrimSpace(sink.Remote) == "" {
				return errors.New("git sink requires remote")
			}
			if strings.TrimSpace(sink.Directory) == "" && strings.TrimSpace(sink.Path) == "" {
				return errors.New("git sink requires directory")
			}
		case "sql":
			if strings.TrimSpace(sink.Driver) == "" {
				return errors.New("sql sink requires driver")
			}
			if strings.TrimSpace(sink.DSN) == "" {
				return errors.New("sql sink requires dsn")
			}
			if strings.TrimSpace(sink.InsertStmt) == "" {
				return errors.New("sql sink requires insert_statement")
			}
			if sink.AutoCreate != nil && *sink.AutoCreate && strings.TrimSpace(sink.CreateStmt) == "" {
				return errors.New("sql sink with auto_create requires create_statement")
			}
		case "otel":
			if hasBothValues(sink.Endpoint, sink.EndpointEnv) {
				return errors.New("otel sink cannot set both endpoint and endpoint_env")
			}
			if len(sink.Headers) > 0 && strings.TrimSpace(sink.HeadersEnv) != "" {
				return errors.New("otel sink cannot set both headers and headers_env")
			}
		default:
			return fmt.Errorf("unsupported sink type %q", sink.Type)
		}
	}

	return nil
}

var allowedFieldsBySinkType = map[string]map[string]bool{
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
		"command": true,
		"args":    true,
		"method":  true,
	},
	"application_insights": {
		"connection_string":     true,
		"connection_string_env": true,
		"event_name":            true,
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

func (s SinkConfig) rejectUnknownFields() error {
	allowed, ok := allowedFieldsBySinkType[s.Type]
	if !ok {
		return fmt.Errorf("unsupported sink type %q", s.Type)
	}

	set := func(field string, isSet bool) error {
		if isSet && !allowed[field] {
			return fmt.Errorf("%s sink does not support field %q", s.Type, field)
		}
		return nil
	}

	checks := []struct {
		field string
		isSet bool
	}{
		{"path", strings.TrimSpace(s.Path) != ""},
		{"directory", strings.TrimSpace(s.Directory) != ""},
		{"format", strings.TrimSpace(s.Format) != "" && s.Format != "markdown"},
		{"command", strings.TrimSpace(s.Command) != ""},
		{"args", len(s.Args) > 0},
		{"method", strings.TrimSpace(s.Method) != ""},
		{"connection_string", strings.TrimSpace(s.ConnectionString) != ""},
		{"connection_string_env", strings.TrimSpace(s.ConnectionStringEnv) != ""},
		{"event_name", strings.TrimSpace(s.EventName) != ""},
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
	for i := range c.Sinks {
		sink := &c.Sinks[i]
		switch strings.ToLower(strings.TrimSpace(sink.Type)) {
		case "application_insights":
			value, err := resolveExclusiveEnvString("application_insights connection string", sink.ConnectionString, sink.ConnectionStringEnv, dotEnv)
			if err != nil {
				return err
			}
			sink.ConnectionString = value
			sink.ConnectionStringEnv = ""
		case "otel":
			value, err := resolveExclusiveEnvString("otel endpoint", sink.Endpoint, sink.EndpointEnv, dotEnv)
			if err != nil {
				return err
			}
			sink.Endpoint = value
			sink.EndpointEnv = ""

			headers, err := resolveExclusiveEnvMap("otel headers", sink.Headers, sink.HeadersEnv, dotEnv)
			if err != nil {
				return err
			}
			sink.Headers = headers
			sink.HeadersEnv = ""
		}
	}
	return nil
}

func resolveExclusiveEnvString(label, directValue, envName string, dotEnv map[string]string) (string, error) {
	if hasBothValues(directValue, envName) {
		return "", fmt.Errorf("%s cannot set both direct value and _env field", label)
	}
	if strings.TrimSpace(envName) == "" {
		return directValue, nil
	}

	value, ok := lookupEnv(strings.TrimSpace(envName), dotEnv)
	if !ok {
		return "", fmt.Errorf("%s env var %q is not set", label, strings.TrimSpace(envName))
	}
	return value, nil
}

func hasBothValues(a, b string) bool {
	return strings.TrimSpace(a) != "" && strings.TrimSpace(b) != ""
}

func resolveExclusiveEnvMap(label string, directValue map[string]string, envName string, dotEnv map[string]string) (map[string]string, error) {
	if len(directValue) > 0 && strings.TrimSpace(envName) != "" {
		return nil, fmt.Errorf("%s cannot set both direct value and _env field", label)
	}
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
		candidate := filepath.Join(current, DefaultFileName)
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}

		parent := filepath.Dir(current)
		if parent == current {
			return "", fmt.Errorf("could not find %s in %s or its parents", DefaultFileName, startDir)
		}
		current = parent
	}
}

func loadDotEnv(dir string) (map[string]string, error) {
	dotEnvPath := filepath.Join(dir, ".env")
	if _, err := os.Stat(dotEnvPath); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("stat .env %s: %w", dotEnvPath, err)
	}
	env, err := godotenv.Read(dotEnvPath)
	if err != nil {
		return nil, fmt.Errorf("read .env %s: %w", dotEnvPath, err)
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
