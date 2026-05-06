package sinks

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/agorischek/token-for-your-thoughts/internal/config"
	"github.com/agorischek/token-for-your-thoughts/internal/feedback"
)

func TestCommandSinkSubmit(t *testing.T) {
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

	t.Setenv("GO_WANT_HELPER_PROCESS", "1")
	if err := sink.Submit(context.Background(), item); err != nil {
		t.Fatalf("submit: %v", err)
	}
}

func TestCommandSinkSubmitJSONRPCError(t *testing.T) {
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

	t.Setenv("GO_WANT_HELPER_PROCESS", "1")
	if err := sink.Submit(context.Background(), item); err == nil {
		t.Fatal("expected submit error")
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

	var request jsonRPCRequest
	if err := json.NewDecoder(os.Stdin).Decode(&request); err != nil {
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

	if err := json.NewEncoder(os.Stdout).Encode(response); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(0)
}
