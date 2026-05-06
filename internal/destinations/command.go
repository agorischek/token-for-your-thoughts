package destinations

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

type CommandDestination struct {
	command string
	args    []string
	method  string

	cmd     *exec.Cmd
	stdin   io.WriteCloser
	decoder *json.Decoder
	stderr  bytes.Buffer

	// exited is closed when the subprocess exits; exitErr holds the result.
	exited  chan struct{}
	exitErr error

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

func NewCommandDestination(cfg config.DestinationConfig) (*CommandDestination, error) {
	if strings.TrimSpace(cfg.Command) == "" {
		return nil, fmt.Errorf("command destination requires command")
	}
	if strings.TrimSpace(cfg.Method) == "" {
		return nil, fmt.Errorf("command destination requires method")
	}

	destination := &CommandDestination{
		command: cfg.Command,
		args:    append([]string(nil), cfg.Args...),
		method:  cfg.Method,
		exited:  make(chan struct{}),
	}

	if err := destination.start(); err != nil {
		return nil, err
	}

	return destination, nil
}

func (s *CommandDestination) Name() string {
	return "command"
}

func (s *CommandDestination) Submit(ctx context.Context, item feedback.Item) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return errors.New("command destination is closed")
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

func (s *CommandDestination) Close(ctx context.Context) error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.closed = true
	stdin := s.stdin
	cmd := s.cmd
	exited := s.exited
	s.mu.Unlock()

	if stdin != nil {
		_ = stdin.Close()
	}
	if cmd == nil {
		return nil
	}

	select {
	case <-exited:
		if s.exitErr != nil {
			return fmt.Errorf("command destination process failed: %w", s.exitErr)
		}
		return nil
	case <-ctx.Done():
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		<-exited
		if s.exitErr != nil {
			return errors.Join(ctx.Err(), fmt.Errorf("command destination process failed: %w", s.exitErr))
		}
		return ctx.Err()
	}
}

func (s *CommandDestination) start() error {
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
		return fmt.Errorf("start command destination process: %w", err)
	}

	go func() {
		_, _ = io.Copy(&s.stderr, stderr)
	}()
	go func() {
		s.exitErr = cmd.Wait()
		close(s.exited)
	}()

	s.cmd = cmd
	s.stdin = stdin
	s.decoder = json.NewDecoder(bufio.NewReader(stdout))
	return nil
}

func (s *CommandDestination) checkExited() error {
	select {
	case <-s.exited:
		if s.exitErr != nil {
			return s.wrapProcessError("command destination process failed", s.exitErr)
		}
		return errors.New("command destination process exited unexpectedly")
	default:
		return nil
	}
}

func (s *CommandDestination) wrapProcessError(prefix string, err error) error {
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
