package sinks

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/agorischek/suggestion-box/internal/config"
	"github.com/agorischek/suggestion-box/internal/feedback"
)

type GitSink struct {
	baseDir       string
	repoRoot      string
	branch        string
	remote        string
	path          string
	push          bool
	commitMessage *template.Template
}

func NewGitSink(baseDir, repoRoot string, cfg config.SinkConfig) (*GitSink, error) {
	if strings.TrimSpace(repoRoot) == "" {
		return nil, fmt.Errorf("git sink requires a git repository")
	}

	commitTemplate, err := template.New("commit-message").Parse(cfg.CommitMessage)
	if err != nil {
		return nil, fmt.Errorf("parse commit message template: %w", err)
	}

	return &GitSink{
		baseDir:       baseDir,
		repoRoot:      repoRoot,
		branch:        cfg.Branch,
		remote:        cfg.Remote,
		path:          cfg.Path,
		push:          cfg.Push != nil && *cfg.Push,
		commitMessage: commitTemplate,
	}, nil
}

func (s *GitSink) Name() string {
	return "git"
}

func (s *GitSink) Submit(ctx context.Context, item feedback.Item) error {
	remoteURL, err := s.gitOutput(ctx, s.repoRoot, "remote", "get-url", s.remote)
	if err != nil {
		return fmt.Errorf("resolve remote %s: %w", s.remote, err)
	}

	tempDir, err := os.MkdirTemp("", "suggestion-box-git-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	if err := s.gitRun(ctx, tempDir, "init"); err != nil {
		return fmt.Errorf("init temp repo: %w", err)
	}
	if err := s.gitRun(ctx, tempDir, "remote", "add", "origin", remoteURL); err != nil {
		return fmt.Errorf("add origin remote: %w", err)
	}
	if err := s.gitRun(ctx, tempDir, "remote", "add", "source", s.repoRoot); err != nil {
		return fmt.Errorf("add source remote: %w", err)
	}

	if err := s.checkoutBranch(ctx, tempDir); err != nil {
		return err
	}

	path := s.path
	if !filepath.IsAbs(path) {
		path = filepath.Join(tempDir, path)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create git sink directory: %w", err)
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open sink file: %w", err)
	}
	if err := ensureHeader(file); err != nil {
		file.Close()
		return err
	}
	if _, err := file.WriteString(item.MarkdownEntry()); err != nil {
		file.Close()
		return fmt.Errorf("append entry: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close sink file: %w", err)
	}

	relPath, err := filepath.Rel(tempDir, path)
	if err != nil {
		return fmt.Errorf("compute relative path: %w", err)
	}

	if err := s.gitRun(ctx, tempDir, "add", relPath); err != nil {
		return fmt.Errorf("stage feedback file: %w", err)
	}

	var commitMessage bytes.Buffer
	if err := s.commitMessage.Execute(&commitMessage, item); err != nil {
		return fmt.Errorf("render commit message: %w", err)
	}
	if err := s.gitRun(ctx, tempDir, "commit", "-m", commitMessage.String()); err != nil {
		return fmt.Errorf("commit feedback entry: %w", err)
	}

	if s.push {
		if err := s.gitRun(ctx, tempDir, "push", "origin", s.branch); err != nil {
			return fmt.Errorf("push feedback branch: %w", err)
		}
	}

	return nil
}

func (s *GitSink) checkoutBranch(ctx context.Context, tempDir string) error {
	if err := s.gitRun(ctx, tempDir, "fetch", "origin", s.branch); err == nil {
		if err := s.gitRun(ctx, tempDir, "checkout", "-b", s.branch, "FETCH_HEAD"); err != nil {
			return fmt.Errorf("checkout remote branch: %w", err)
		}
		return nil
	}

	head, err := s.gitOutput(ctx, s.repoRoot, "rev-parse", "HEAD")
	if err != nil {
		return fmt.Errorf("resolve local HEAD: %w", err)
	}
	if err := s.gitRun(ctx, tempDir, "fetch", "source", head); err != nil {
		return fmt.Errorf("fetch local HEAD: %w", err)
	}
	if err := s.gitRun(ctx, tempDir, "checkout", "-b", s.branch, "FETCH_HEAD"); err != nil {
		return fmt.Errorf("create feedback branch: %w", err)
	}
	return nil
}

func (s *GitSink) gitRun(ctx context.Context, dir string, args ...string) error {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %s", strings.Join(args, " "), strings.TrimSpace(string(output)))
	}
	return nil
}

func (s *GitSink) gitOutput(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s: %s", strings.Join(args, " "), strings.TrimSpace(string(output)))
	}
	return strings.TrimSpace(string(output)), nil
}

func ensureHeader(file *os.File) error {
	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("stat sink file: %w", err)
	}
	if info.Size() != 0 {
		return nil
	}
	if _, err := file.WriteString("# Feedback\n\n"); err != nil {
		return fmt.Errorf("write header: %w", err)
	}
	return nil
}
