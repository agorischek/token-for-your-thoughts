package app

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/agorischek/suggestion-box/internal/config"
	"github.com/agorischek/suggestion-box/internal/feedback"
	"github.com/agorischek/suggestion-box/internal/mcpserver"
	"github.com/agorischek/suggestion-box/internal/sinks"
)

type runtimeConfig struct {
	Config     config.Config
	ConfigPath string
	BaseDir    string
	RepoRoot   string
	Manager    *sinks.Manager
}

func Run(ctx context.Context, version string, args []string, stdout, stderr io.Writer) error {
	if len(args) == 0 {
		printHelp(stdout)
		return nil
	}

	switch args[0] {
	case "help", "-h", "--help":
		printHelp(stdout)
		return nil
	case "version", "--version":
		_, err := fmt.Fprintf(stdout, "%s\n", version)
		return err
	case "submit":
		return runSubmit(ctx, args[1:], stdout, stderr)
	case "serve-mcp":
		return runServeMCP(ctx, version, args[1:], stderr)
	default:
		printHelp(stderr)
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func runSubmit(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("submit", flag.ContinueOnError)
	fs.SetOutput(stderr)

	var configPath string
	var provider string
	var summary string
	var details string
	var category string
	var source string
	var metadataRaw string

	fs.StringVar(&configPath, "config", "", "path to a .suggestionsrc file")
	fs.StringVar(&provider, "provider", "", "feedback provider, for example Claude Code")
	fs.StringVar(&summary, "summary", "", "short summary of the feedback")
	fs.StringVar(&details, "details", "", "longer description of the feedback")
	fs.StringVar(&category, "category", "", "optional category such as tooling or instructions")
	fs.StringVar(&source, "source", "cli", "origin of the submission")
	fs.StringVar(&metadataRaw, "metadata", "", "optional JSON object with extra metadata")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() > 0 {
		return errors.New("submit does not accept positional arguments")
	}

	runtime, err := loadRuntimeConfig(configPath)
	if err != nil {
		return err
	}

	metadata, err := parseMetadata(metadataRaw)
	if err != nil {
		return err
	}

	item, err := feedback.New(provider, summary, details, category, source, metadata)
	if err != nil {
		return err
	}

	result, err := runtime.Manager.Submit(ctx, item)
	if err != nil {
		return err
	}

	if _, err := fmt.Fprintf(stdout, "submitted feedback %s\n", item.ID); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(stdout, "successful sinks: %s\n", strings.Join(result.Succeeded, ", ")); err != nil {
		return err
	}
	if len(result.Failed) > 0 {
		failed := make([]string, 0, len(result.Failed))
		for name, msg := range result.Failed {
			failed = append(failed, fmt.Sprintf("%s (%s)", name, msg))
		}
		sort.Strings(failed)
		if _, err := fmt.Fprintf(stdout, "failed sinks: %s\n", strings.Join(failed, ", ")); err != nil {
			return err
		}
	}

	return nil
}

func runServeMCP(ctx context.Context, version string, args []string, stderr io.Writer) error {
	fs := flag.NewFlagSet("serve-mcp", flag.ContinueOnError)
	fs.SetOutput(stderr)

	var configPath string
	fs.StringVar(&configPath, "config", "", "path to a .suggestionsrc file")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() > 0 {
		return errors.New("serve-mcp does not accept positional arguments")
	}

	runtime, err := loadRuntimeConfig(configPath)
	if err != nil {
		return err
	}

	return mcpserver.Serve(ctx, version, runtime.Config, runtime.Manager)
}

func loadRuntimeConfig(explicitPath string) (*runtimeConfig, error) {
	cfg, resolvedPath, err := config.Load(explicitPath, ".")
	if err != nil {
		return nil, err
	}

	baseDir := filepath.Dir(resolvedPath)
	repoRoot, err := gitRoot(baseDir)
	if err != nil {
		if needsGitSink(cfg) {
			return nil, err
		}
		repoRoot = ""
	}

	manager, err := sinks.NewManager(cfg, baseDir, repoRoot)
	if err != nil {
		return nil, err
	}

	return &runtimeConfig{
		Config:     cfg,
		ConfigPath: resolvedPath,
		BaseDir:    baseDir,
		RepoRoot:   repoRoot,
		Manager:    manager,
	}, nil
}

func gitRoot(dir string) (string, error) {
	cmd := exec.Command("git", "-C", dir, "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		return "", errors.New("git sink requires running inside a git repository")
	}
	return strings.TrimSpace(string(output)), nil
}

func needsGitSink(cfg config.Config) bool {
	for _, sink := range cfg.Sinks {
		if strings.EqualFold(sink.Type, "git") {
			return true
		}
	}
	return false
}

func parseMetadata(raw string) (map[string]any, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}

	var metadata map[string]any
	if err := json.Unmarshal([]byte(raw), &metadata); err != nil {
		return nil, fmt.Errorf("parse metadata: %w", err)
	}
	return metadata, nil
}

func printHelp(w io.Writer) {
	fmt.Fprint(w, `suggestion-box collects agent feedback for a repository.

Usage:
  suggestion-box submit --provider "Claude Code" --summary "..." [flags]
  suggestion-box serve-mcp [flags]
  suggestion-box version

Commands:
  submit      Submit feedback directly from the CLI
  serve-mcp   Serve the MCP submit_feedback tool over stdio
  version     Print the build version

Config:
  suggestion-box loads .suggestionsrc JSON from the current directory or the
  nearest parent directory unless --config is provided.
`)
}
