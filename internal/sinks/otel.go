package sinks

import (
	"context"
	"fmt"
	"strings"

	"github.com/agorischek/suggestion-box/internal/config"
	"github.com/agorischek/suggestion-box/internal/feedback"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

type OTelSink struct {
	name     string
	provider *sdktrace.TracerProvider
	tracer   trace.Tracer
}

func NewOTelSink(ctx context.Context, cfg config.SinkConfig) (*OTelSink, error) {
	options := make([]otlptracehttp.Option, 0, 3)
	if endpoint := strings.TrimSpace(cfg.Endpoint); endpoint != "" {
		if strings.HasPrefix(endpoint, "http://") || strings.HasPrefix(endpoint, "https://") {
			options = append(options, otlptracehttp.WithEndpointURL(endpoint))
		} else {
			options = append(options, otlptracehttp.WithEndpoint(endpoint))
		}
	}
	if cfg.Insecure {
		options = append(options, otlptracehttp.WithInsecure())
	}
	if len(cfg.Headers) > 0 {
		options = append(options, otlptracehttp.WithHeaders(cfg.Headers))
	}

	exporter, err := otlptracehttp.New(ctx, options...)
	if err != nil {
		return nil, fmt.Errorf("create otel exporter: %w", err)
	}

	res, err := resource.New(ctx, resource.WithAttributes(
		semconv.ServiceNameKey.String(cfg.ServiceName),
		attribute.String("suggestion_box.sink", "otel"),
	))
	if err != nil {
		return nil, fmt.Errorf("create otel resource: %w", err)
	}

	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)

	return &OTelSink{
		name:     "otel",
		provider: provider,
		tracer:   provider.Tracer("github.com/agorischek/suggestion-box"),
	}, nil
}

func (s *OTelSink) Name() string {
	return s.name
}

func (s *OTelSink) Submit(ctx context.Context, item feedback.Item) error {
	_, span := s.tracer.Start(ctx, "suggestion-box.feedback")
	span.SetAttributes(
		attribute.String("feedback.id", item.ID),
		attribute.String("feedback.provider", item.Provider),
		attribute.String("feedback.summary", item.Summary),
		attribute.String("feedback.details", item.Details),
		attribute.String("feedback.category", item.Category),
		attribute.String("feedback.source", item.Source),
		attribute.String("feedback.created_at", item.CreatedAt.Format("2006-01-02T15:04:05Z07:00")),
		attribute.String("feedback.metadata_json", item.MetadataJSON()),
	)
	span.End()

	if err := s.provider.ForceFlush(ctx); err != nil {
		return fmt.Errorf("flush otel spans: %w", err)
	}
	return nil
}
