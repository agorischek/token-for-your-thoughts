package destinations

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

type HTTPDestination struct {
	client          *http.Client
	url             string
	method          string
	headers         map[string]string
	successStatuses map[int]bool
}

func NewHTTPDestination(cfg config.DestinationConfig) (*HTTPDestination, error) {
	return newHTTPDestination(cfg, &http.Client{Timeout: time.Duration(cfg.TimeoutSeconds) * time.Second})
}

func newHTTPDestination(cfg config.DestinationConfig, client *http.Client) (*HTTPDestination, error) {
	headers := map[string]string{}
	for key, value := range cfg.Headers {
		headers[key] = value
	}

	successStatuses := map[int]bool{}
	for _, status := range cfg.SuccessStatuses {
		successStatuses[status] = true
	}

	return &HTTPDestination{
		client:          client,
		url:             cfg.URL,
		method:          cfg.Method,
		headers:         headers,
		successStatuses: successStatuses,
	}, nil
}

func (s *HTTPDestination) Name() string {
	return "http"
}

func (s *HTTPDestination) Submit(ctx context.Context, item feedback.Item) error {
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

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1 MB cap
	if err != nil {
		return fmt.Errorf("read http response: %w", err)
	}

	if s.isSuccessStatus(resp.StatusCode) {
		return nil
	}

	message := strings.TrimSpace(string(respBody))
	if message == "" {
		return fmt.Errorf("http destination returned %s", resp.Status)
	}
	return fmt.Errorf("http destination returned %s: %s", resp.Status, message)
}

func (s *HTTPDestination) isSuccessStatus(status int) bool {
	if len(s.successStatuses) == 0 {
		return status >= 200 && status < 300
	}
	return s.successStatuses[status]
}
