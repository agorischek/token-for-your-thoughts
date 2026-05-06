package sinks

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/agorischek/token-for-your-thoughts/internal/config"
	"github.com/agorischek/token-for-your-thoughts/internal/feedback"
)

func TestApplicationInsightsSinkSubmit(t *testing.T) {
	t.Parallel()

	var gotPath string
	var gotBody []applicationInsightsEnvelope
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"itemsReceived":1,"itemsAccepted":1}`))
	}))
	defer server.Close()

	sink, err := newApplicationInsightsSink(config.SinkConfig{
		Type:             "application_insights",
		ConnectionString: "InstrumentationKey=00000000-0000-0000-0000-000000000000;IngestionEndpoint=" + server.URL + "/",
		EventName:        "repo feedback",
	}, server.Client())
	if err != nil {
		t.Fatalf("new sink: %v", err)
	}

	item, err := feedback.New("Claude Code", "The Application Insights sink should send custom events.", "cli", map[string]any{"team": "agents"})
	if err != nil {
		t.Fatalf("new item: %v", err)
	}

	if err := sink.Submit(context.Background(), item); err != nil {
		t.Fatalf("submit: %v", err)
	}

	if gotPath != "/v2/track" {
		t.Fatalf("unexpected path %q", gotPath)
	}
	if len(gotBody) != 1 {
		t.Fatalf("expected 1 envelope, got %d", len(gotBody))
	}
	envelope := gotBody[0]
	if envelope.IKey != "00000000-0000-0000-0000-000000000000" {
		t.Fatalf("unexpected ikey %q", envelope.IKey)
	}
	if envelope.Data.BaseData.Name != "repo feedback" {
		t.Fatalf("unexpected event name %q", envelope.Data.BaseData.Name)
	}
	if envelope.Data.BaseData.Properties["feedback.provider"] != "Claude Code" {
		t.Fatalf("missing provider property")
	}
	if !strings.Contains(envelope.Data.BaseData.Properties["feedback.metadata_json"], "agents") {
		t.Fatalf("missing metadata json")
	}
}

func TestParseConnectionString(t *testing.T) {
	t.Parallel()

	parts, err := parseConnectionString("InstrumentationKey=abc;EndpointSuffix=applicationinsights.azure.com")
	if err != nil {
		t.Fatalf("parse connection string: %v", err)
	}
	if parts["InstrumentationKey"] != "abc" {
		t.Fatalf("unexpected instrumentation key %q", parts["InstrumentationKey"])
	}
}
