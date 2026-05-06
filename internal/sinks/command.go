package sinks

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/agorischek/token-for-your-thoughts/internal/config"
	"github.com/agorischek/token-for-your-thoughts/internal/feedback"
)

type CommandSink struct {
	command string
	args    []string
	method  string
}

type jsonRPCRequest struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      string        `json:"id"`
	Method  string        `json:"method"`
	Params  feedback.Item `json:"params"`
}

type jsonRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      string          `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *jsonRPCError   `json:"error,omitempty"`
}

type jsonRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func NewCommandSink(cfg config.SinkConfig) (*CommandSink, error) {
	if strings.TrimSpace(cfg.Command) == "" {
		return nil, fmt.Errorf("command sink requires command")
	}
	if strings.TrimSpace(cfg.Method) == "" {
		return nil, fmt.Errorf("command sink requires method")
	}

	return &CommandSink{
		command: cfg.Command,
		args:    append([]string(nil), cfg.Args...),
		method:  cfg.Method,
	}, nil
}

func (s *CommandSink) Name() string {
	return "command"
}

func (s *CommandSink) Submit(ctx context.Context, item feedback.Item) error {
	cmd := exec.CommandContext(ctx, s.command, s.args...)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("open command stdin: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("open command stdout: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("open command stderr: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start command sink process: %w", err)
	}

	request := jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      item.ID,
		Method:  s.method,
		Params:  item,
	}

	writeErr := writeJSONRPCRequest(stdin, request)
	stderrData, stderrErr := io.ReadAll(stderr)
	response, readErr := readJSONRPCResponse(stdout)
	waitErr := cmd.Wait()

	if writeErr != nil {
		return fmt.Errorf("write json-rpc request: %w", writeErr)
	}
	if stderrErr != nil {
		return fmt.Errorf("read command stderr: %w", stderrErr)
	}
	if readErr != nil {
		if len(stderrData) > 0 {
			return fmt.Errorf("read json-rpc response: %w: %s", readErr, strings.TrimSpace(string(stderrData)))
		}
		return fmt.Errorf("read json-rpc response: %w", readErr)
	}
	if waitErr != nil {
		if len(stderrData) > 0 {
			return fmt.Errorf("command sink process failed: %w: %s", waitErr, strings.TrimSpace(string(stderrData)))
		}
		return fmt.Errorf("command sink process failed: %w", waitErr)
	}
	if response.JSONRPC != "2.0" {
		return fmt.Errorf("unexpected json-rpc version %q", response.JSONRPC)
	}
	if response.ID != item.ID {
		return fmt.Errorf("unexpected json-rpc response id %q", response.ID)
	}
	if response.Error != nil {
		return fmt.Errorf("json-rpc error %d: %s", response.Error.Code, response.Error.Message)
	}

	return nil
}

func writeJSONRPCRequest(w io.WriteCloser, request jsonRPCRequest) error {
	defer w.Close()

	encoder := json.NewEncoder(w)
	return encoder.Encode(request)
}

func readJSONRPCResponse(r io.Reader) (jsonRPCResponse, error) {
	var response jsonRPCResponse
	decoder := json.NewDecoder(bufio.NewReader(r))
	if err := decoder.Decode(&response); err != nil {
		return jsonRPCResponse{}, err
	}
	return response, nil
}
