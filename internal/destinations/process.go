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

type ProcessDestination struct {
	command string
	args    []string
	method  string

	cmd     *exec.Cmd
	stdin   io.WriteCloser
	stdout  io.ReadCloser
	decoder *json.Decoder
	stderr  bytes.Buffer

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

func NewProcessDestination(cfg config.DestinationConfig) (*ProcessDestination, error) {
	if strings.TrimSpace(cfg.Command) == "" {
		return nil, fmt.Errorf("process destination requires command")
	}
	if strings.TrimSpace(cfg.Method) == "" {
		return nil, fmt.Errorf("process destination requires method")
	}

	destination := &ProcessDestination{
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

func (d *ProcessDestination) Name() string {
	return "process"
}

func (d *ProcessDestination) Submit(ctx context.Context, item feedback.Item) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.closed {
		return errors.New("process destination is closed")
	}
	if err := d.checkExited(); err != nil {
		return err
	}

	request := jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      item.ID,
		Method:  d.method,
		Params:  item,
	}

	if err := writeJSONRPCRequest(d.stdin, request); err != nil {
		return fmt.Errorf("write json-rpc request: %w", err)
	}

	response, err := readJSONRPCResponse(ctx, d.decoder, d.stdout)
	if err != nil {
		return d.wrapProcessError("read json-rpc response", err)
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

func (d *ProcessDestination) Close(ctx context.Context) error {
	d.mu.Lock()
	if d.closed {
		d.mu.Unlock()
		return nil
	}
	d.closed = true
	stdin := d.stdin
	cmd := d.cmd
	exited := d.exited
	d.mu.Unlock()

	if stdin != nil {
		_ = stdin.Close()
	}
	if cmd == nil {
		return nil
	}

	select {
	case <-exited:
		if d.exitErr != nil {
			return fmt.Errorf("process destination process failed: %w", d.exitErr)
		}
		return nil
	case <-ctx.Done():
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		<-exited
		if d.exitErr != nil {
			return errors.Join(ctx.Err(), fmt.Errorf("process destination process failed: %w", d.exitErr))
		}
		return ctx.Err()
	}
}

func (d *ProcessDestination) start() error {
	cmd := exec.Command(d.command, d.args...)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("open process stdin: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("open process stdout: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("open process stderr: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start process destination process: %w", err)
	}

	go func() {
		_, _ = io.Copy(&d.stderr, stderr)
	}()
	go func() {
		d.exitErr = cmd.Wait()
		close(d.exited)
	}()

	d.cmd = cmd
	d.stdin = stdin
	d.stdout = stdout
	d.decoder = json.NewDecoder(bufio.NewReader(stdout))
	return nil
}

func (d *ProcessDestination) checkExited() error {
	select {
	case <-d.exited:
		if d.exitErr != nil {
			return d.wrapProcessError("process destination process failed", d.exitErr)
		}
		return errors.New("process destination process exited unexpectedly")
	default:
		return nil
	}
}

func (d *ProcessDestination) wrapProcessError(prefix string, err error) error {
	stderr := strings.TrimSpace(d.stderr.String())
	if stderr != "" {
		return fmt.Errorf("%s: %w: %s", prefix, err, stderr)
	}
	return fmt.Errorf("%s: %w", prefix, err)
}

func writeJSONRPCRequest(w io.Writer, request jsonRPCRequest) error {
	encoder := json.NewEncoder(w)
	return encoder.Encode(request)
}

func readJSONRPCResponse(ctx context.Context, decoder *json.Decoder, stdout io.Closer) (jsonRPCResponse, error) {
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
		_ = stdout.Close()
		<-done
		return jsonRPCResponse{}, ctx.Err()
	}
}
