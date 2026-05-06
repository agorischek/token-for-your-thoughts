package sinks

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/agorischek/token-for-your-thoughts/internal/config"
	"github.com/agorischek/token-for-your-thoughts/internal/feedback"
)

type Sink interface {
	Name() string
	Submit(context.Context, feedback.Item) error
}

type managedSink interface {
	Sink
	Close(context.Context) error
}

type Result struct {
	Succeeded []string
	Failed    map[string]string
}

type Manager struct {
	sinks []Sink
}

func NewManager(ctx context.Context, cfg config.Config, baseDir, repoRoot string) (*Manager, error) {
	created := make([]Sink, 0, len(cfg.Sinks))

	for _, sinkConfig := range cfg.Sinks {
		var sink Sink
		var err error

		switch sinkConfig.Type {
		case "file":
			sink, err = NewFileSink(baseDir, sinkConfig)
		case "http":
			sink, err = NewHTTPSink(sinkConfig)
		case "command":
			sink, err = NewCommandSink(sinkConfig)
		case "application_insights":
			sink, err = NewApplicationInsightsSink(sinkConfig)
		case "git":
			sink, err = NewGitSink(baseDir, repoRoot, sinkConfig)
		case "sql":
			sink, err = NewSQLSink(baseDir, sinkConfig)
		case "otel":
			sink, err = NewOTelSink(ctx, sinkConfig)
		default:
			err = fmt.Errorf("unsupported sink type %q", sinkConfig.Type)
		}
		if err != nil {
			return nil, err
		}
		created = append(created, sink)
	}

	return &Manager{sinks: created}, nil
}

func (m *Manager) Close(ctx context.Context) error {
	var errs []error
	for _, s := range m.sinks {
		if sink, ok := s.(managedSink); ok {
			errs = append(errs, sink.Close(ctx))
		}
	}
	return errors.Join(errs...)
}

func (m *Manager) Submit(ctx context.Context, item feedback.Item) (Result, error) {
	result := Result{
		Succeeded: make([]string, 0, len(m.sinks)),
		Failed:    map[string]string{},
	}
	var errs []error

	for _, sink := range m.sinks {
		if err := sink.Submit(ctx, item); err != nil {
			result.Failed[sink.Name()] = err.Error()
			errs = append(errs, fmt.Errorf("%s: %w", sink.Name(), err))
			continue
		}
		result.Succeeded = append(result.Succeeded, sink.Name())
	}

	sort.Strings(result.Succeeded)
	if len(result.Failed) == 0 {
		return result, nil
	}
	if len(result.Succeeded) == 0 {
		return result, errors.Join(errs...)
	}
	return result, nil
}
