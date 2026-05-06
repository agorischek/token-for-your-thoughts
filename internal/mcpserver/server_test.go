package mcpserver

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/agorischek/token-for-your-thoughts/internal/config"
	"github.com/agorischek/token-for-your-thoughts/internal/destinations"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestServeRegistersToolAndAcceptsSubmission(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfg := config.Config{
		MCP: config.MCPConfig{
			ToolName:        "submit_feedback",
			ToolDescription: "Submit feedback",
		},
		Destinations: []config.DestinationConfig{
			{Type: "file", Path: filepath.Join(dir, "FEEDBACK.md"), Format: "markdown"},
		},
	}

	manager, err := destinations.NewManager(context.Background(), cfg, dir, "")
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	defer manager.Close(context.Background())

	server := mcp.NewServer(&mcp.Implementation{
		Name:    "tfyt",
		Version: "test",
	}, nil)

	registerTool(server, cfg, manager)

	serverTransport, clientTransport := mcp.NewInMemoryTransports()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	serverDone := make(chan error, 1)
	go func() {
		serverDone <- server.Run(ctx, serverTransport)
	}()

	client := mcp.NewClient(&mcp.Implementation{
		Name:    "test-client",
		Version: "1.0.0",
	}, nil)

	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("connect client: %v", err)
	}
	defer session.Close()

	tools, err := session.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}
	if len(tools.Tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(tools.Tools))
	}
	if tools.Tools[0].Name != "submit_feedback" {
		t.Fatalf("unexpected tool name %q", tools.Tools[0].Name)
	}

	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "submit_feedback",
		Arguments: map[string]any{
			"provider": "TestAgent",
			"feedback": "MCP integration test feedback",
		},
	})
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	if result.IsError {
		t.Fatalf("tool returned error: %+v", result.Content)
	}

	data, err := os.ReadFile(filepath.Join(dir, "FEEDBACK.md"))
	if err != nil {
		t.Fatalf("read feedback file: %v", err)
	}
	if !strings.Contains(string(data), "MCP integration test feedback") {
		t.Fatalf("feedback file missing message: %s", string(data))
	}
}

func TestServeReturnsErrorForMissingProvider(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfg := config.Config{
		MCP: config.MCPConfig{
			ToolName:        "submit_feedback",
			ToolDescription: "Submit feedback",
		},
		Destinations: []config.DestinationConfig{
			{Type: "file", Path: filepath.Join(dir, "FEEDBACK.md"), Format: "markdown"},
		},
	}

	manager, err := destinations.NewManager(context.Background(), cfg, dir, "")
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	defer manager.Close(context.Background())

	server := mcp.NewServer(&mcp.Implementation{
		Name:    "tfyt",
		Version: "test",
	}, nil)

	registerTool(server, cfg, manager)

	serverTransport, clientTransport := mcp.NewInMemoryTransports()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		_ = server.Run(ctx, serverTransport)
	}()

	client := mcp.NewClient(&mcp.Implementation{
		Name:    "test-client",
		Version: "1.0.0",
	}, nil)

	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("connect client: %v", err)
	}
	defer session.Close()

	result, err := session.CallTool(ctx, &mcp.CallToolParams{
		Name: "submit_feedback",
		Arguments: map[string]any{
			"feedback": "missing provider",
		},
	})
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected tool error for missing provider")
	}
}

func TestToolErrorFormatsMessage(t *testing.T) {
	t.Parallel()

	result := toolError(context.DeadlineExceeded)
	if !result.IsError {
		t.Fatal("expected IsError to be true")
	}
	if len(result.Content) != 1 {
		t.Fatalf("expected 1 content item, got %d", len(result.Content))
	}
	text, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", result.Content[0])
	}
	if text.Text != "context deadline exceeded" {
		t.Fatalf("unexpected error text %q", text.Text)
	}
}
