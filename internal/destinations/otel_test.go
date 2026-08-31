package destinations

import (
	"context"
	"testing"

	"github.com/agorischek/token-for-your-thoughts/internal/feedback"
	"go.opentelemetry.io/otel/attribute"
	otellog "go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/embedded"
)

type recordingLogger struct {
	embedded.Logger

	records []otellog.Record
}

func (l *recordingLogger) Emit(_ context.Context, record otellog.Record) {
	l.records = append(l.records, record)
}

func (l *recordingLogger) Enabled(context.Context, otellog.EnabledParameters) bool {
	return true
}

func TestOTelDestinationSubmitRecord(t *testing.T) {
	t.Parallel()

	logger := &recordingLogger{}
	destination := &OTelDestination{name: "otel", logger: logger}

	item, err := feedback.New("Claude Code", "The OTel destination should emit a log record.", "cli", map[string]any{"team": "agents"})
	if err != nil {
		t.Fatalf("new item: %v", err)
	}

	if err := destination.Submit(context.Background(), item); err != nil {
		t.Fatalf("submit: %v", err)
	}

	if len(logger.records) != 1 {
		t.Fatalf("unexpected record count %d", len(logger.records))
	}

	record := logger.records[0]
	if record.EventName() != "tfyt.feedback" {
		t.Fatalf("unexpected event name %q", record.EventName())
	}
	if body := record.Body(); body.AsString() != item.Feedback {
		t.Fatalf("unexpected body %q", body.AsString())
	}

	attrs := make(map[attribute.Key]string, record.AttributesLen())
	record.WalkAttributes(func(kv attribute.KeyValue) bool {
		attrs[kv.Key] = kv.Value.AsString()
		return true
	})

	expected := map[attribute.Key]string{
		"feedback.id":            item.ID,
		"feedback.provider":      item.Provider,
		"feedback.source":        item.Source,
		"feedback.metadata_json": item.MetadataJSON(),
	}
	for key, want := range expected {
		if got := attrs[key]; got != want {
			t.Fatalf("unexpected attribute %q: got %q, want %q", key, got, want)
		}
	}
	if _, ok := attrs["feedback.created_at"]; !ok {
		t.Fatal("missing feedback.created_at attribute")
	}
}
