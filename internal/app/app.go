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

	"github.com/agorischek/token-for-your-thoughts/internal/config"
	"github.com/agorischek/token-for-your-thoughts/internal/destinations"
	"github.com/agorischek/token-for-your-thoughts/internal/feedback"
	"github.com/agorischek/token-for-your-thoughts/internal/mcpserver"
)

type runtimeConfig struct {
	Config     config.Config
	ConfigPath string
	BaseDir    string
	RepoRoot   string
	Manager    *destinations.Manager
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
	var feedbackText string
	var source string
	var metadataRaw string

	fs.StringVar(&configPath, "config", "", "path to a tfyt.toml or tfyt.json file")
	fs.StringVar(&provider, "provider", "", "feedback provider, for example Claude Code")
	fs.StringVar(&feedbackText, "feedback", "", "feedback text")
	fs.StringVar(&source, "source", "cli", "origin of the submission")
	fs.StringVar(&metadataRaw, "metadata", "", "optional JSON object with extra metadata")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() > 0 {
		return errors.New("submit does not accept positional arguments")
	}

	runtime, err := loadRuntimeConfig(ctx, configPath)
	if err != nil {
		return err
	}
	defer runtime.Manager.Close(ctx)

	metadata, err := parseMetadata(metadataRaw)
	if err != nil {
		return err
	}

	item, err := feedback.New(provider, feedbackText, source, metadata)
	if err != nil {
		return err
	}

	result, submitErr := runtime.Manager.Submit(ctx, item)

	if _, err := fmt.Fprintf(stdout, "submitted feedback %s\n", item.ID); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(stdout, "successful destinations: %s\n", strings.Join(result.Succeeded, ", ")); err != nil {
		return err
	}
	if len(result.Failed) > 0 {
		failed := make([]string, 0, len(result.Failed))
		for name, msg := range result.Failed {
			failed = append(failed, fmt.Sprintf("%s (%s)", name, msg))
		}
		sort.Strings(failed)
		if _, err := fmt.Fprintf(stdout, "failed destinations: %s\n", strings.Join(failed, ", ")); err != nil {
			return err
		}
	}

	return submitErr
}

func runServeMCP(ctx context.Context, version string, args []string, stderr io.Writer) error {
	fs := flag.NewFlagSet("serve-mcp", flag.ContinueOnError)
	fs.SetOutput(stderr)

	var configPath string
	fs.StringVar(&configPath, "config", "", "path to a tfyt.toml or tfyt.json file")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() > 0 {
		return errors.New("serve-mcp does not accept positional arguments")
	}

	runtime, err := loadRuntimeConfig(ctx, configPath)
	if err != nil {
		return err
	}
	defer runtime.Manager.Close(ctx)

	return mcpserver.Serve(ctx, version, runtime.Config, runtime.Manager)
}

func loadRuntimeConfig(ctx context.Context, explicitPath string) (*runtimeConfig, error) {
	cfg, resolvedPath, err := config.Load(explicitPath, ".")
	if err != nil {
		return nil, err
	}

	baseDir := filepath.Dir(resolvedPath)
	repoRoot, err := gitRoot(baseDir)
	if err != nil {
		if needsGitDestination(cfg) {
			return nil, err
		}
		repoRoot = ""
	}

	manager, err := destinations.NewManager(ctx, cfg, baseDir, repoRoot)
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
		return "", errors.New("git destination requires running inside a git repository")
	}
	return strings.TrimSpace(string(output)), nil
}

func needsGitDestination(cfg config.Config) bool {
	for _, destination := range cfg.Destinations {
		if strings.EqualFold(destination.Type, "git") {
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
	fmt.Fprint(w, `tfyt collects agent feedback for a repository.

Usage:
  tfyt submit --provider "Claude Code" --feedback "..." [flags]
  tfyt serve-mcp [flags]
  tfyt version

Commands:
  submit      Submit feedback directly from the CLI
  serve-mcp   Serve the MCP submit_feedback tool over stdio
  version     Print the build version

Submit flags:
  --provider    Feedback provider name (required)
  --feedback    Feedback text (required)
  --source      Origin of the submission (default: cli)
  --metadata    Optional JSON object with extra metadata
  --config      Path to a tfyt.toml or tfyt.json file

Config:
  tfyt loads tfyt.toml first, then tfyt.json, from the current
  directory or the nearest parent directory unless --config is provided.
`)
}
