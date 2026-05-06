package destinations

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/agorischek/token-for-your-thoughts/internal/config"
	"github.com/agorischek/token-for-your-thoughts/internal/feedback"
)

type CommandDestination struct {
	command     string
	args        []string
	contentMode string
}

func NewCommandDestination(cfg config.DestinationConfig) (*CommandDestination, error) {
	if strings.TrimSpace(cfg.Command) == "" {
		return nil, fmt.Errorf("command destination requires command")
	}
	if cfg.ContentMode != "json" && cfg.ContentMode != "args" {
		return nil, fmt.Errorf("command destination content_mode must be json or args")
	}

	return &CommandDestination{
		command:     cfg.Command,
		args:        append([]string(nil), cfg.Args...),
		contentMode: cfg.ContentMode,
	}, nil
}

func (d *CommandDestination) Name() string {
	return "command"
}

func (d *CommandDestination) Submit(ctx context.Context, item feedback.Item) error {
	args := append([]string(nil), d.args...)
	cmd := exec.CommandContext(ctx, d.command)

	switch d.contentMode {
	case "args":
		args = append(args, feedbackArgs(item)...)
		cmd.Args = append([]string{d.command}, args...)
	default:
		cmd.Args = append([]string{d.command}, args...)
		data, err := item.JSON(true)
		if err != nil {
			return fmt.Errorf("marshal feedback json: %w", err)
		}
		cmd.Stdin = strings.NewReader(string(append(data, '\n')))
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		trimmed := strings.TrimSpace(string(output))
		if trimmed != "" {
			return fmt.Errorf("run command destination: %w: %s", err, trimmed)
		}
		return fmt.Errorf("run command destination: %w", err)
	}
	return nil
}

func feedbackArgs(item feedback.Item) []string {
	args := []string{
		"--id", item.ID,
		"--provider", item.Provider,
		"--feedback", item.Feedback,
		"--source", item.Source,
		"--created-at", item.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}

	if len(item.Metadata) > 0 {
		metadataJSON, err := json.Marshal(item.Metadata)
		if err == nil {
			args = append(args, "--metadata-json", string(metadataJSON))
		}
	}

	return args
}
