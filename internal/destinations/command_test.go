package destinations

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/agorischek/token-for-your-thoughts/internal/config"
	"github.com/agorischek/token-for-your-thoughts/internal/feedback"
)

func TestCommandDestinationSubmitJSON(t *testing.T) {
	t.Setenv("GO_WANT_HELPER_PROCESS", "1")

	destination, err := NewCommandDestination(config.DestinationConfig{
		Type:        "command",
		Command:     os.Args[0],
		Args:        []string{"-test.run=TestHelperProcessCommandDestination", "--", "json-ok"},
		ContentMode: "json",
	})
	if err != nil {
		t.Fatalf("new command destination: %v", err)
	}

	item, err := feedback.New("Claude Code", "The command destination should send JSON over stdin.", "cli", map[string]any{"tool": "test"})
	if err != nil {
		t.Fatalf("new item: %v", err)
	}

	if err := destination.Submit(context.Background(), item); err != nil {
		t.Fatalf("submit: %v", err)
	}
}

func TestCommandDestinationSubmitArgs(t *testing.T) {
	t.Setenv("GO_WANT_HELPER_PROCESS", "1")

	destination, err := NewCommandDestination(config.DestinationConfig{
		Type:        "command",
		Command:     os.Args[0],
		Args:        []string{"-test.run=TestHelperProcessCommandDestination", "--", "args-ok"},
		ContentMode: "args",
	})
	if err != nil {
		t.Fatalf("new command destination: %v", err)
	}

	item, err := feedback.New("Claude Code", "The command destination should send feedback as CLI flags.", "cli", map[string]any{"tool": "test"})
	if err != nil {
		t.Fatalf("new item: %v", err)
	}

	if err := destination.Submit(context.Background(), item); err != nil {
		t.Fatalf("submit: %v", err)
	}
}

func TestCommandDestinationIsOneShot(t *testing.T) {
	t.Setenv("GO_WANT_HELPER_PROCESS", "1")

	counterFile := filepathJoin(t.TempDir(), "starts.txt")
	t.Setenv("GO_HELPER_START_COUNT_FILE", counterFile)

	destination, err := NewCommandDestination(config.DestinationConfig{
		Type:        "command",
		Command:     os.Args[0],
		Args:        []string{"-test.run=TestHelperProcessCommandDestination", "--", "json-ok"},
		ContentMode: "json",
	})
	if err != nil {
		t.Fatalf("new command destination: %v", err)
	}

	first, _ := feedback.New("Claude Code", "First one-shot.", "cli", nil)
	second, _ := feedback.New("Claude Code", "Second one-shot.", "cli", nil)

	if err := destination.Submit(context.Background(), first); err != nil {
		t.Fatalf("submit first: %v", err)
	}
	if err := destination.Submit(context.Background(), second); err != nil {
		t.Fatalf("submit second: %v", err)
	}

	countBytes, err := os.ReadFile(counterFile)
	if err != nil {
		t.Fatalf("read counter file: %v", err)
	}
	count, err := strconv.Atoi(string(countBytes))
	if err != nil {
		t.Fatalf("parse counter file: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected helper process to start twice, got %d", count)
	}
}

func TestCommandDestinationReportsFailure(t *testing.T) {
	t.Setenv("GO_WANT_HELPER_PROCESS", "1")

	destination, err := NewCommandDestination(config.DestinationConfig{
		Type:        "command",
		Command:     os.Args[0],
		Args:        []string{"-test.run=TestHelperProcessCommandDestination", "--", "fail"},
		ContentMode: "json",
	})
	if err != nil {
		t.Fatalf("new command destination: %v", err)
	}

	item, err := feedback.New("Claude Code", "The helper should reject this request.", "cli", nil)
	if err != nil {
		t.Fatalf("new item: %v", err)
	}

	if err := destination.Submit(context.Background(), item); err == nil {
		t.Fatal("expected submit error")
	}
}

func TestHelperProcessCommandDestination(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	mode := "json-ok"
	for i, arg := range os.Args {
		if arg == "--" && i+1 < len(os.Args) {
			mode = os.Args[i+1]
			break
		}
	}

	if counterFile := os.Getenv("GO_HELPER_START_COUNT_FILE"); counterFile != "" {
		count := 0
		if raw, err := os.ReadFile(counterFile); err == nil {
			count, _ = strconv.Atoi(string(raw))
		}
		count++
		if err := os.WriteFile(counterFile, []byte(strconv.Itoa(count)), 0o644); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}

	switch mode {
	case "json-ok":
		var item feedback.Item
		if err := json.NewDecoder(os.Stdin).Decode(&item); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if item.Provider == "" || item.Feedback == "" {
			fmt.Fprintln(os.Stderr, "missing feedback payload")
			os.Exit(1)
		}
		os.Exit(0)
	case "args-ok":
		parsed := parseHelperArgs(os.Args)
		if parsed["provider"] == "" || parsed["feedback"] == "" || parsed["created-at"] == "" {
			fmt.Fprintln(os.Stderr, "missing expected feedback args")
			os.Exit(1)
		}
		if parsed["metadata-json"] == "" || !strings.Contains(parsed["metadata-json"], `"tool":"test"`) {
			fmt.Fprintln(os.Stderr, "missing metadata json arg")
			os.Exit(1)
		}
		os.Exit(0)
	case "fail":
		io.Copy(io.Discard, os.Stdin)
		fmt.Fprintln(os.Stderr, "helper rejected feedback")
		os.Exit(2)
	default:
		fmt.Fprintln(os.Stderr, "unknown mode")
		os.Exit(1)
	}
}

func parseHelperArgs(args []string) map[string]string {
	values := map[string]string{}
	for i := 0; i < len(args); i++ {
		if !strings.HasPrefix(args[i], "--") || i+1 >= len(args) {
			continue
		}
		values[strings.TrimPrefix(args[i], "--")] = args[i+1]
		i++
	}
	return values
}

func filepathJoin(parts ...string) string {
	return strings.Join(parts, string(os.PathSeparator))
}
