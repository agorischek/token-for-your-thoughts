package destinations

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/agorischek/token-for-your-thoughts/internal/config"
	"github.com/agorischek/token-for-your-thoughts/internal/feedback"
)

func TestHTTPSinkSubmit(t *testing.T) {
	t.Parallel()

	var gotMethod string
	var gotContentType string
	var gotAuth string
	var gotItem feedback.Item

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotContentType = r.Header.Get("Content-Type")
		gotAuth = r.Header.Get("Authorization")
		if err := json.NewDecoder(r.Body).Decode(&gotItem); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	destination, err := newHTTPDestination(config.DestinationConfig{
		Type:            "http",
		URL:             server.URL,
		Method:          http.MethodPost,
		Headers:         map[string]string{"Authorization": "Bearer test-token"},
		TimeoutSeconds:  10,
		SuccessStatuses: []int{http.StatusAccepted},
	}, server.Client())
	if err != nil {
		t.Fatalf("new destination: %v", err)
	}

	item, err := feedback.New("Claude Code", "The HTTP destination should send the full feedback item as JSON.", "cli", map[string]any{"team": "agents"})
	if err != nil {
		t.Fatalf("new item: %v", err)
	}

	if err := destination.Submit(context.Background(), item); err != nil {
		t.Fatalf("submit: %v", err)
	}

	if gotMethod != http.MethodPost {
		t.Fatalf("unexpected method %q", gotMethod)
	}
	if gotContentType != "application/json" {
		t.Fatalf("unexpected content type %q", gotContentType)
	}
	if gotAuth != "Bearer test-token" {
		t.Fatalf("unexpected authorization header %q", gotAuth)
	}
	if gotItem.ID != item.ID {
		t.Fatalf("unexpected id %q", gotItem.ID)
	}
	if gotItem.Provider != "Claude Code" {
		t.Fatalf("unexpected provider %q", gotItem.Provider)
	}
	if gotItem.Feedback != item.Feedback {
		t.Fatalf("unexpected feedback %q", gotItem.Feedback)
	}
	if gotItem.Metadata["team"] != "agents" {
		t.Fatalf("unexpected metadata %#v", gotItem.Metadata)
	}
}

func TestHTTPSinkDefaultsTo2xx(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	destination, err := newHTTPDestination(config.DestinationConfig{
		Type:           "http",
		URL:            server.URL,
		Method:         http.MethodPost,
		TimeoutSeconds: 10,
	}, server.Client())
	if err != nil {
		t.Fatalf("new destination: %v", err)
	}

	item, err := feedback.New("Claude Code", "A 204 response should count as success.", "cli", nil)
	if err != nil {
		t.Fatalf("new item: %v", err)
	}

	if err := destination.Submit(context.Background(), item); err != nil {
		t.Fatalf("submit: %v", err)
	}
}

func TestHTTPSinkReturnsResponseBodyOnError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "webhook rejected payload", http.StatusBadGateway)
	}))
	defer server.Close()

	destination, err := newHTTPDestination(config.DestinationConfig{
		Type:           "http",
		URL:            server.URL,
		Method:         http.MethodPost,
		TimeoutSeconds: 10,
	}, server.Client())
	if err != nil {
		t.Fatalf("new destination: %v", err)
	}

	item, err := feedback.New("Claude Code", "The HTTP destination should surface response bodies on failure.", "cli", nil)
	if err != nil {
		t.Fatalf("new item: %v", err)
	}

	err = destination.Submit(context.Background(), item)
	if err == nil {
		t.Fatal("expected submit error")
	}
	if got := err.Error(); got == "" || got == "http destination returned 502 Bad Gateway" {
		t.Fatalf("expected detailed error, got %q", got)
	}
}

func TestHTTPSinkUsesClientTimeout(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	destination, err := NewHTTPDestination(config.DestinationConfig{
		Type:           "http",
		URL:            server.URL,
		Method:         http.MethodPost,
		TimeoutSeconds: 1,
	})
	if err != nil {
		t.Fatalf("new destination: %v", err)
	}

	if destination.client.Timeout != time.Second {
		t.Fatalf("unexpected client timeout %s", destination.client.Timeout)
	}
}
