package sinks

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/agorischek/suggesting/internal/config"
	"github.com/agorischek/suggesting/internal/feedback"
	"go.opentelemetry.io/otel/attribute"
	otellog "go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

type OTelSink struct {
	name     string
	provider *sdklog.LoggerProvider
	logger   otellog.Logger
}

func NewOTelSink(ctx context.Context, cfg config.SinkConfig) (*OTelSink, error) {
	options := make([]otlploghttp.Option, 0, 3)
	if endpoint := strings.TrimSpace(cfg.Endpoint); endpoint != "" {
		if strings.HasPrefix(endpoint, "http://") || strings.HasPrefix(endpoint, "https://") {
			options = append(options, otlploghttp.WithEndpointURL(endpoint))
		} else {
			options = append(options, otlploghttp.WithEndpoint(endpoint))
		}
	}
	if cfg.Insecure {
		options = append(options, otlploghttp.WithInsecure())
	}
	if len(cfg.Headers) > 0 {
		options = append(options, otlploghttp.WithHeaders(cfg.Headers))
	}

	exporter, err := otlploghttp.New(ctx, options...)
	if err != nil {
		return nil, fmt.Errorf("create otel exporter: %w", err)
	}

	res, err := resource.New(ctx, resource.WithAttributes(
		semconv.ServiceNameKey.String(cfg.ServiceName),
		attribute.String("suggesting.sink", "otel"),
	))
	if err != nil {
		return nil, fmt.Errorf("create otel resource: %w", err)
	}

	provider := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewBatchProcessor(exporter)),
		sdklog.WithResource(res),
	)

	return &OTelSink{
		name:     "otel",
		provider: provider,
		logger:   provider.Logger("github.com/agorischek/suggesting"),
	}, nil
}

func (s *OTelSink) Name() string {
	return s.name
}

func (s *OTelSink) Close(ctx context.Context) error {
	return s.provider.Shutdown(ctx)
}

func (s *OTelSink) Submit(ctx context.Context, item feedback.Item) error {
	record := otellog.Record{}
	record.SetTimestamp(item.CreatedAt)
	record.SetObservedTimestamp(time.Now().UTC())
	record.SetBody(otellog.StringValue(item.Feedback))
	record.SetEventName("suggesting.feedback")
	record.AddAttributes(
		otellog.String("feedback.id", item.ID),
		otellog.String("feedback.provider", item.Provider),
		otellog.String("feedback.source", item.Source),
		otellog.String("feedback.created_at", item.CreatedAt.Format(time.RFC3339)),
		otellog.String("feedback.metadata_json", item.MetadataJSON()),
	)

	s.logger.Emit(ctx, record)

	if err := s.provider.ForceFlush(ctx); err != nil {
		return fmt.Errorf("flush otel logs: %w", err)
	}
	return nil
}
