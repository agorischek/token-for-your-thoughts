package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	DefaultFileName                     = ".suggesting.json"
	defaultMCPToolName                  = "submit_feedback"
	defaultMCPDescription               = "Submit feedback about working in this repository, including tool errors and inefficiencies, as well as instruction gaps and inconsistencies."
	defaultFeedbackPath                 = "FEEDBACK.md"
	defaultFeedbackJSONPath             = "FEEDBACK.jsonl"
	defaultGitRemote                    = "origin"
	defaultGitBranch                    = "feedback"
	defaultGitDirectory                 = ".feedback"
	defaultGitCommitPrefix              = "Add feedback entry"
	defaultCommandMethod                = "submit_feedback"
	defaultApplicationInsightsEventName = "suggesting feedback"
	defaultSQLTable                     = "feedback"
	defaultOTelServiceName              = "suggesting"
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

	Path             string            `json:"path,omitempty"`
	Directory        string            `json:"directory,omitempty"`
	Format           string            `json:"format,omitempty"`
	Command          string            `json:"command,omitempty"`
	Args             []string          `json:"args,omitempty"`
	Method           string            `json:"method,omitempty"`
	ConnectionString string            `json:"connection_string,omitempty"`
	EventName        string            `json:"event_name,omitempty"`
	Branch           string            `json:"branch,omitempty"`
	Remote           string            `json:"remote,omitempty"`
	Push             *bool             `json:"push,omitempty"`
	CommitMessage    string            `json:"commit_message,omitempty"`
	Driver           string            `json:"driver,omitempty"`
	DSN              string            `json:"dsn,omitempty"`
	Table            string            `json:"table,omitempty"`
	AutoCreate       *bool             `json:"auto_create,omitempty"`
	CreateStmt       string            `json:"create_statement,omitempty"`
	InsertStmt       string            `json:"insert_statement,omitempty"`
	Endpoint         string            `json:"endpoint,omitempty"`
	Headers          map[string]string `json:"headers,omitempty"`
	Insecure         bool              `json:"insecure,omitempty"`
	ServiceName      string            `json:"service_name,omitempty"`
}

func Load(explicitPath, startDir string) (Config, string, error) {
	path, err := locate(explicitPath, startDir)
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
	if len(c.Sinks) == 0 {
		return errors.New("config must define at least one sink")
	}

	for _, sink := range c.Sinks {
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
			if strings.TrimSpace(sink.ConnectionString) == "" {
				return errors.New("application_insights sink requires connection_string")
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
		default:
			return fmt.Errorf("unsupported sink type %q", sink.Type)
		}
	}

	return nil
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
