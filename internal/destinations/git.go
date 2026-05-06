package destinations

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/agorischek/token-for-your-thoughts/internal/config"
	"github.com/agorischek/token-for-your-thoughts/internal/feedback"
)

type GitDestination struct {
	baseDir       string
	repoRoot      string
	branch        string
	remote        string
	directory     string
	format        string
	push          bool
	commitMessage *template.Template
}

func NewGitDestination(baseDir, repoRoot string, cfg config.DestinationConfig) (*GitDestination, error) {
	if strings.TrimSpace(repoRoot) == "" {
		return nil, fmt.Errorf("git destination requires a git repository")
	}

	commitTemplate, err := template.New("commit-message").Parse(cfg.CommitMessage)
	if err != nil {
		return nil, fmt.Errorf("parse commit message template: %w", err)
	}

	return &GitDestination{
		baseDir:       baseDir,
		repoRoot:      repoRoot,
		branch:        cfg.Branch,
		remote:        cfg.Remote,
		directory:     cfg.Directory,
		format:        cfg.Format,
		push:          cfg.Push != nil && *cfg.Push,
		commitMessage: commitTemplate,
	}, nil
}

func (s *GitDestination) Name() string {
	return "git"
}

func (s *GitDestination) Submit(ctx context.Context, item feedback.Item) error {
	remoteURL, err := s.gitOutput(ctx, s.repoRoot, "remote", "get-url", s.remote)
	if err != nil {
		return fmt.Errorf("resolve remote %s: %w", s.remote, err)
	}

	tempDir, err := os.MkdirTemp("", "tfyt-git-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	if err := s.gitRun(ctx, tempDir, "init"); err != nil {
		return fmt.Errorf("init temp repo: %w", err)
	}
	if err := s.gitRun(ctx, tempDir, "config", "user.name", "tfyt"); err != nil {
		return fmt.Errorf("set git user.name: %w", err)
	}
	if err := s.gitRun(ctx, tempDir, "config", "user.email", "tfyt@noreply"); err != nil {
		return fmt.Errorf("set git user.email: %w", err)
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

	if filepath.IsAbs(s.directory) {
		return fmt.Errorf("git destination directory must be relative to the repository root")
	}

	directoryPath := filepath.Join(tempDir, filepath.Clean(s.directory))
	relDirectory, err := filepath.Rel(tempDir, directoryPath)
	if err != nil {
		return fmt.Errorf("compute git destination directory: %w", err)
	}
	if relDirectory == ".." || strings.HasPrefix(relDirectory, ".."+string(filepath.Separator)) {
		return fmt.Errorf("git destination directory must stay within the repository root")
	}
	if err := os.MkdirAll(directoryPath, 0o755); err != nil {
		return fmt.Errorf("create git destination directory: %w", err)
	}

	extension := ".md"
	var content []byte
	switch s.format {
	case "json":
		extension = ".json"
		content, err = item.JSON(true)
		if err != nil {
			return fmt.Errorf("marshal feedback json: %w", err)
		}
		content = append(content, '\n')
	default:
		content = []byte(item.MarkdownEntry())
	}

	path := filepath.Join(directoryPath, item.ID+extension)
	if err := os.WriteFile(path, content, 0o644); err != nil {
		return fmt.Errorf("write feedback file: %w", err)
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

func (s *GitDestination) checkoutBranch(ctx context.Context, tempDir string) error {
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

func (s *GitDestination) gitRun(ctx context.Context, dir string, args ...string) error {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %s", strings.Join(args, " "), strings.TrimSpace(string(output)))
	}
	return nil
}

func (s *GitDestination) gitOutput(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s: %s", strings.Join(args, " "), strings.TrimSpace(string(output)))
	}
	return strings.TrimSpace(string(output)), nil
}
