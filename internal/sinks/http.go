package sinks

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/agorischek/token-for-your-thoughts/internal/config"
	"github.com/agorischek/token-for-your-thoughts/internal/feedback"
)

type HTTPSink struct {
	client          *http.Client
	url             string
	method          string
	headers         map[string]string
	successStatuses map[int]bool
}

func NewHTTPSink(cfg config.SinkConfig) (*HTTPSink, error) {
	return newHTTPSink(cfg, &http.Client{Timeout: time.Duration(cfg.TimeoutSeconds) * time.Second})
}

func newHTTPSink(cfg config.SinkConfig, client *http.Client) (*HTTPSink, error) {
	headers := map[string]string{}
	for key, value := range cfg.Headers {
		headers[key] = value
	}

	successStatuses := map[int]bool{}
	for _, status := range cfg.SuccessStatuses {
		successStatuses[status] = true
	}

	return &HTTPSink{
		client:          client,
		url:             cfg.URL,
		method:          cfg.Method,
		headers:         headers,
		successStatuses: successStatuses,
	}, nil
}

func (s *HTTPSink) Name() string {
	return "http"
}

func (s *HTTPSink) Submit(ctx context.Context, item feedback.Item) error {
	body, err := item.JSON(false)
	if err != nil {
		return fmt.Errorf("marshal http payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, s.method, s.url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create http request: %w", err)
	}
	for key, value := range s.headers {
		req.Header.Set(key, value)
	}
	if req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("send http feedback: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read http response: %w", err)
	}

	if s.isSuccessStatus(resp.StatusCode) {
		return nil
	}

	message := strings.TrimSpace(string(respBody))
	if message == "" {
		return fmt.Errorf("http sink returned %s", resp.Status)
	}
	return fmt.Errorf("http sink returned %s: %s", resp.Status, message)
}

func (s *HTTPSink) isSuccessStatus(status int) bool {
	if len(s.successStatuses) == 0 {
		return status >= 200 && status < 300
	}
	return s.successStatuses[status]
}
