package mcpserver

import (
	"context"
	"fmt"
	"strings"

	"github.com/agorischek/suggestion-box/internal/config"
	"github.com/agorischek/suggestion-box/internal/feedback"
	"github.com/agorischek/suggestion-box/internal/sinks"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type submitInput struct {
	Provider string         `json:"provider" jsonschema:"Provider name for the agent or tool, for example Claude Code"`
	Summary  string         `json:"summary" jsonschema:"Short summary of the feedback"`
	Details  string         `json:"details,omitempty" jsonschema:"Longer description with context, repro steps, or examples"`
	Category string         `json:"category,omitempty" jsonschema:"Optional category such as tooling, instructions, workflow, or ergonomics"`
	Metadata map[string]any `json:"metadata,omitempty" jsonschema:"Optional structured metadata"`
}

type submitOutput struct {
	ID        string            `json:"id" jsonschema:"GUID assigned to the feedback"`
	CreatedAt string            `json:"created_at" jsonschema:"RFC3339 timestamp for when the feedback was recorded"`
	Succeeded []string          `json:"succeeded" jsonschema:"Sinks that accepted the feedback"`
	Failed    map[string]string `json:"failed,omitempty" jsonschema:"Sinks that returned an error"`
}

func Serve(ctx context.Context, version string, cfg config.Config, manager *sinks.Manager) error {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "suggestion-box",
		Version: version,
	}, nil)

	mcp.AddTool(server, &mcp.Tool{
		Name:        cfg.ToolName(),
		Description: cfg.ToolDescription(),
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input submitInput) (*mcp.CallToolResult, submitOutput, error) {
		item, err := feedback.New(input.Provider, input.Summary, input.Details, input.Category, "mcp", input.Metadata)
		if err != nil {
			return toolError(err), submitOutput{}, nil
		}

		result, err := manager.Submit(ctx, item)
		output := submitOutput{
			ID:        item.ID,
			CreatedAt: item.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			Succeeded: result.Succeeded,
			Failed:    result.Failed,
		}

		if err != nil {
			return toolError(fmt.Errorf("feedback %s: %w", item.ID, err)), output, nil
		}

		return nil, output, nil
	})

	return server.Run(ctx, &mcp.StdioTransport{})
}

func toolError(err error) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{
			&mcp.TextContent{Text: strings.TrimSpace(err.Error())},
		},
	}
}
