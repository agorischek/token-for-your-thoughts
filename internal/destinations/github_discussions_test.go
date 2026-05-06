package destinations

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"text/template"
	"time"

	"github.com/agorischek/token-for-your-thoughts/internal/config"
	"github.com/agorischek/token-for-your-thoughts/internal/feedback"
	"github.com/google/go-github/v74/github"
)

func TestGitHubDiscussionsDestinationCreatesDiscussion(t *testing.T) {
	t.Parallel()

	var categoriesCalls atomic.Int32
	var createCalls atomic.Int32
	var seenAuthorization string
	var seenTitle string
	var seenBody string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/graphql" {
			http.NotFound(w, r)
			return
		}

		seenAuthorization = r.Header.Get("Authorization")

		var payload struct {
			Query     string         `json:"query"`
			Variables map[string]any `json:"variables"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request: %v", err)
		}

		switch {
		case strings.Contains(payload.Query, "discussionCategories"):
			categoriesCalls.Add(1)
			io.WriteString(w, `{"data":{"repository":{"id":"R_kgDOTestRepo","discussionCategories":{"nodes":[{"id":"DIC_kwDOTestCat","name":"Feedback","slug":"feedback"}]}}}}`)
		case strings.Contains(payload.Query, "createDiscussion"):
			createCalls.Add(1)
			input, ok := payload.Variables["input"].(map[string]any)
			if !ok {
				t.Fatalf("expected input variables map, got %#v", payload.Variables["input"])
			}
			seenTitle, _ = input["title"].(string)
			seenBody, _ = input["body"].(string)
			io.WriteString(w, `{"data":{"createDiscussion":{"discussion":{"id":"D_kwDOTestDiscussion"}}}}`)
		default:
			t.Fatalf("unexpected query: %s", payload.Query)
		}
	}))
	defer server.Close()

	client := github.NewClient(server.Client()).WithAuthToken("test-token")
	baseURL, err := url.Parse(server.URL + "/")
	if err != nil {
		t.Fatalf("parse base url: %v", err)
	}
	client.BaseURL = baseURL

	cfg := config.DestinationConfig{
		Type:          "github_discussions",
		Repository:    "octo/example",
		Category:      "feedback",
		TitleTemplate: "Feedback {{ .ID }} from {{ .Provider }}",
	}
	titleTemplate, err := template.New("discussion-title").Parse(cfg.TitleTemplate)
	if err != nil {
		t.Fatalf("parse title template: %v", err)
	}

	destination := newGitHubDiscussionsDestination(client, cfg, "octo", "example", titleTemplate)
	item := feedback.Item{
		ID:        "d0d08407-c381-43ab-bd9e-a74f3c0224ef",
		Provider:  "GitHub Copilot CLI",
		Feedback:  "Test feedback from Copilot CLI via MCP. Do not use shell commands or edit files.",
		Source:    "mcp",
		CreatedAt: time.Date(2026, 5, 6, 2, 1, 59, 0, time.UTC),
		Metadata: map[string]any{
			"source": "mcp",
		},
	}

	if err := destination.Submit(context.Background(), item); err != nil {
		t.Fatalf("submit first discussion: %v", err)
	}
	if err := destination.Submit(context.Background(), item); err != nil {
		t.Fatalf("submit second discussion: %v", err)
	}

	if categoriesCalls.Load() != 1 {
		t.Fatalf("expected one category lookup, got %d", categoriesCalls.Load())
	}
	if createCalls.Load() != 2 {
		t.Fatalf("expected two discussion creations, got %d", createCalls.Load())
	}
	if seenAuthorization != "Bearer test-token" {
		t.Fatalf("unexpected authorization header %q", seenAuthorization)
	}
	if seenTitle != "Feedback d0d08407-c381-43ab-bd9e-a74f3c0224ef from GitHub Copilot CLI" {
		t.Fatalf("unexpected discussion title %q", seenTitle)
	}
	for _, expected := range []string{
		"Test feedback from Copilot CLI via MCP. Do not use shell commands or edit files.",
		"_From GitHub Copilot CLI via MCP at 2026-05-06T02:01:59Z_",
		`"source": "mcp"`,
	} {
		if !strings.Contains(seenBody, expected) {
			t.Fatalf("discussion body missing %q", expected)
		}
	}
}

func TestGitHubDiscussionsDestinationRejectsInvalidRepository(t *testing.T) {
	t.Parallel()

	_, err := NewGitHubDiscussionsDestination(config.DestinationConfig{
		Type:          "github_discussions",
		Repository:    "not-a-slug",
		Category:      "feedback",
		Token:         "token",
		TitleTemplate: "Feedback {{ .ID }}",
	})
	if err == nil {
		t.Fatal("expected invalid repository error")
	}
}

func TestGitHubDiscussionsDestinationErrorsWhenCategoryIsMissing(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"data":{"repository":{"id":"R_kgDOTestRepo","discussionCategories":{"nodes":[{"id":"DIC_kwDOOther","name":"General","slug":"general"}]}}}}`)
	}))
	defer server.Close()

	client := github.NewClient(server.Client()).WithAuthToken("test-token")
	baseURL, err := url.Parse(server.URL + "/")
	if err != nil {
		t.Fatalf("parse base url: %v", err)
	}
	client.BaseURL = baseURL

	cfg := config.DestinationConfig{
		Type:          "github_discussions",
		Repository:    "octo/example",
		Category:      "feedback",
		TitleTemplate: "Feedback {{ .ID }}",
	}
	titleTemplate, err := template.New("discussion-title").Parse(cfg.TitleTemplate)
	if err != nil {
		t.Fatalf("parse title template: %v", err)
	}

	destination := newGitHubDiscussionsDestination(client, cfg, "octo", "example", titleTemplate)
	item := feedback.Item{
		ID:        "97059329-7216-4b46-8cf3-b21223114f3f",
		Provider:  "Codex",
		Feedback:  "config validation smoke",
		Source:    "cli",
		CreatedAt: time.Date(2026, 5, 6, 1, 59, 30, 0, time.UTC),
	}

	if err := destination.Submit(context.Background(), item); err == nil {
		t.Fatal("expected missing category error")
	}
}
