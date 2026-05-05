package mcpserver

import (
	"context"
	"fmt"
	"strings"

	"github.com/agorischek/suggesting/internal/config"
	"github.com/agorischek/suggesting/internal/feedback"
	"github.com/agorischek/suggesting/internal/sinks"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type submitInput struct {
	Provider string         `json:"provider" jsonschema:"Provider name for the agent or tool, for example Claude Code"`
	Feedback string         `json:"feedback" jsonschema:"The feedback itself, including any context, repro steps, or examples"`
	Metadata map[string]any `json:"metadata,omitempty" jsonschema:"Optional structured metadata"`
}

type submitOutput struct {
	ID        string            `json:"id" jsonschema:"GUID assigned to the feedback"`
	Succeeded []string          `json:"succeeded" jsonschema:"Sinks that accepted the feedback"`
	Failed    map[string]string `json:"failed,omitempty" jsonschema:"Sinks that returned an error"`
}

func Serve(ctx context.Context, version string, cfg config.Config, manager *sinks.Manager) error {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "suggesting",
		Version: version,
	}, nil)

	mcp.AddTool(server, &mcp.Tool{
		Name:        cfg.ToolName(),
		Description: cfg.ToolDescription(),
	}, func(ctx context.Context, _ *mcp.CallToolRequest, input submitInput) (*mcp.CallToolResult, submitOutput, error) {
		item, err := feedback.New(input.Provider, input.Feedback, "mcp", input.Metadata)
		if err != nil {
			return toolError(err), submitOutput{}, nil
		}

		result, err := manager.Submit(ctx, item)
		output := submitOutput{
			ID:        item.ID,
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
