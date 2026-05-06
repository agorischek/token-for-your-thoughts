package sinks

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/agorischek/token-for-your-thoughts/internal/config"
	"github.com/agorischek/token-for-your-thoughts/internal/feedback"
)

func TestCommandSinkSubmit(t *testing.T) {
	t.Setenv("GO_WANT_HELPER_PROCESS", "1")

	sink, err := NewCommandSink(config.SinkConfig{
		Type:    "command",
		Command: os.Args[0],
		Args:    []string{"-test.run=TestHelperProcessCommandSink", "--", "ok"},
		Method:  "submit_feedback",
	})
	if err != nil {
		t.Fatalf("new command sink: %v", err)
	}

	item, err := feedback.New("Claude Code", "The command sink should deliver JSON-RPC over stdio.", "cli", map[string]any{"tool": "test"})
	if err != nil {
		t.Fatalf("new item: %v", err)
	}

	if err := sink.Submit(context.Background(), item); err != nil {
		t.Fatalf("submit: %v", err)
	}
	if err := sink.Close(context.Background()); err != nil {
		t.Fatalf("close: %v", err)
	}
}

func TestCommandSinkSubmitJSONRPCError(t *testing.T) {
	t.Setenv("GO_WANT_HELPER_PROCESS", "1")

	sink, err := NewCommandSink(config.SinkConfig{
		Type:    "command",
		Command: os.Args[0],
		Args:    []string{"-test.run=TestHelperProcessCommandSink", "--", "error"},
		Method:  "submit_feedback",
	})
	if err != nil {
		t.Fatalf("new command sink: %v", err)
	}

	item, err := feedback.New("Claude Code", "The helper should reject this request.", "cli", nil)
	if err != nil {
		t.Fatalf("new item: %v", err)
	}

	if err := sink.Submit(context.Background(), item); err == nil {
		t.Fatal("expected submit error")
	}
	if err := sink.Close(context.Background()); err != nil {
		t.Fatalf("close: %v", err)
	}
}

func TestCommandSinkReusesProcess(t *testing.T) {
	t.Setenv("GO_WANT_HELPER_PROCESS", "1")

	counterFile := filepath.Join(t.TempDir(), "starts.txt")
	t.Setenv("GO_HELPER_START_COUNT_FILE", counterFile)

	sink, err := NewCommandSink(config.SinkConfig{
		Type:    "command",
		Command: os.Args[0],
		Args:    []string{"-test.run=TestHelperProcessCommandSink", "--", "ok"},
		Method:  "submit_feedback",
	})
	if err != nil {
		t.Fatalf("new command sink: %v", err)
	}

	first, err := feedback.New("Claude Code", "First submission should keep the helper alive.", "mcp", nil)
	if err != nil {
		t.Fatalf("new first item: %v", err)
	}
	second, err := feedback.New("Claude Code", "Second submission should reuse the helper process.", "mcp", nil)
	if err != nil {
		t.Fatalf("new second item: %v", err)
	}

	if err := sink.Submit(context.Background(), first); err != nil {
		t.Fatalf("submit first: %v", err)
	}
	if err := sink.Submit(context.Background(), second); err != nil {
		t.Fatalf("submit second: %v", err)
	}
	if err := sink.Close(context.Background()); err != nil {
		t.Fatalf("close: %v", err)
	}

	countBytes, err := os.ReadFile(counterFile)
	if err != nil {
		t.Fatalf("read counter file: %v", err)
	}
	count, err := strconv.Atoi(string(countBytes))
	if err != nil {
		t.Fatalf("parse counter file: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected helper process to start once, got %d", count)
	}
}

func TestHelperProcessCommandSink(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	mode := "ok"
	for i, arg := range os.Args {
		if arg == "--" && i+1 < len(os.Args) {
			mode = os.Args[i+1]
			break
		}
	}

	if counterFile := os.Getenv("GO_HELPER_START_COUNT_FILE"); counterFile != "" {
		if err := os.WriteFile(counterFile, []byte("1"), 0o644); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}

	decoder := json.NewDecoder(os.Stdin)
	encoder := json.NewEncoder(os.Stdout)
	for {
		var request jsonRPCRequest
		if err := decoder.Decode(&request); err != nil {
			if err == io.EOF {
				os.Exit(0)
			}
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		if request.JSONRPC != "2.0" || request.Method != "submit_feedback" || request.Params.Provider == "" || request.Params.Feedback == "" {
			fmt.Fprintln(os.Stderr, "unexpected request payload")
			os.Exit(1)
		}

		response := jsonRPCResponse{
			JSONRPC: "2.0",
			ID:      request.ID,
			Result:  json.RawMessage(`{"ok":true}`),
		}
		if mode == "error" {
			response.Error = &jsonRPCError{Code: -32000, Message: "sink rejected feedback"}
			response.Result = nil
		}

		if err := encoder.Encode(response); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
}
