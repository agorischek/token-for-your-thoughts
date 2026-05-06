package sinks

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"

	"github.com/agorischek/token-for-your-thoughts/internal/config"
	"github.com/agorischek/token-for-your-thoughts/internal/feedback"
)

type CommandSink struct {
	command string
	args    []string
	method  string

	cmd        *exec.Cmd
	stdin      io.WriteCloser
	decoder    *json.Decoder
	stderr     bytes.Buffer
	waitResult chan error

	mu     sync.Mutex
	closed bool
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

	sink := &CommandSink{
		command:    cfg.Command,
		args:       append([]string(nil), cfg.Args...),
		method:     cfg.Method,
		waitResult: make(chan error, 1),
	}

	if err := sink.start(); err != nil {
		return nil, err
	}

	return sink, nil
}

func (s *CommandSink) Name() string {
	return "command"
}

func (s *CommandSink) Submit(ctx context.Context, item feedback.Item) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return errors.New("command sink is closed")
	}
	if err := s.checkExited(); err != nil {
		return err
	}

	request := jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      item.ID,
		Method:  s.method,
		Params:  item,
	}

	if err := writeJSONRPCRequest(s.stdin, request); err != nil {
		return fmt.Errorf("write json-rpc request: %w", err)
	}

	response, err := readJSONRPCResponse(ctx, s.decoder)
	if err != nil {
		return s.wrapProcessError("read json-rpc response", err)
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

func (s *CommandSink) Close(ctx context.Context) error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.closed = true
	stdin := s.stdin
	cmd := s.cmd
	waitResult := s.waitResult
	s.mu.Unlock()

	if stdin != nil {
		_ = stdin.Close()
	}
	if cmd == nil {
		return nil
	}

	select {
	case err := <-waitResult:
		if err != nil {
			return fmt.Errorf("command sink process failed: %w", err)
		}
		return nil
	case <-ctx.Done():
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		err := <-waitResult
		if err != nil {
			return errors.Join(ctx.Err(), fmt.Errorf("command sink process failed: %w", err))
		}
		return ctx.Err()
	}
}

func (s *CommandSink) start() error {
	cmd := exec.Command(s.command, s.args...)

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

	go func() {
		_, _ = io.Copy(&s.stderr, stderr)
	}()
	go func() {
		s.waitResult <- cmd.Wait()
	}()

	s.cmd = cmd
	s.stdin = stdin
	s.decoder = json.NewDecoder(bufio.NewReader(stdout))
	return nil
}

func (s *CommandSink) checkExited() error {
	select {
	case err := <-s.waitResult:
		if err != nil {
			return s.wrapProcessError("command sink process failed", err)
		}
		return errors.New("command sink process exited unexpectedly")
	default:
		return nil
	}
}

func (s *CommandSink) wrapProcessError(prefix string, err error) error {
	stderr := strings.TrimSpace(s.stderr.String())
	if stderr != "" {
		return fmt.Errorf("%s: %w: %s", prefix, err, stderr)
	}
	return fmt.Errorf("%s: %w", prefix, err)
}

func writeJSONRPCRequest(w io.Writer, request jsonRPCRequest) error {
	encoder := json.NewEncoder(w)
	return encoder.Encode(request)
}

func readJSONRPCResponse(ctx context.Context, decoder *json.Decoder) (jsonRPCResponse, error) {
	type result struct {
		response jsonRPCResponse
		err      error
	}

	done := make(chan result, 1)
	go func() {
		var response jsonRPCResponse
		err := decoder.Decode(&response)
		done <- result{response: response, err: err}
	}()

	select {
	case result := <-done:
		if result.err != nil {
			return jsonRPCResponse{}, result.err
		}
		return result.response, nil
	case <-ctx.Done():
		return jsonRPCResponse{}, ctx.Err()
	}
}
