package destinations

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/agorischek/token-for-your-thoughts/internal/config"
	"github.com/agorischek/token-for-your-thoughts/internal/feedback"
)

type Destination interface {
	Name() string
	Submit(context.Context, feedback.Item) error
}

type managedDestination interface {
	Destination
	Close(context.Context) error
}

type Result struct {
	Succeeded []string
	Failed    map[string]string
}

type Manager struct {
	destinations []Destination
}

func NewManager(ctx context.Context, cfg config.Config, baseDir, repoRoot string) (*Manager, error) {
	created := make([]Destination, 0, len(cfg.Destinations))

	for _, destinationConfig := range cfg.Destinations {
		var destination Destination
		var err error

		switch destinationConfig.Type {
		case "file":
			destination, err = NewFileDestination(baseDir, destinationConfig)
		case "http":
			destination, err = NewHTTPDestination(destinationConfig)
		case "command":
			destination, err = NewCommandDestination(destinationConfig)
		case "application_insights":
			destination, err = NewApplicationInsightsDestination(destinationConfig)
		case "git":
			destination, err = NewGitDestination(baseDir, repoRoot, destinationConfig)
		case "sql":
			destination, err = NewSQLDestination(baseDir, destinationConfig)
		case "otel":
			destination, err = NewOTelDestination(ctx, destinationConfig)
		default:
			err = fmt.Errorf("unsupported destination type %q", destinationConfig.Type)
		}
		if err != nil {
			return nil, err
		}
		created = append(created, destination)
	}

	return &Manager{destinations: created}, nil
}

func (m *Manager) Close(ctx context.Context) error {
	var errs []error
	for _, destination := range m.destinations {
		if managed, ok := destination.(managedDestination); ok {
			errs = append(errs, managed.Close(ctx))
		}
	}
	return errors.Join(errs...)
}

func (m *Manager) Submit(ctx context.Context, item feedback.Item) (Result, error) {
	result := Result{
		Succeeded: make([]string, 0, len(m.destinations)),
		Failed:    map[string]string{},
	}
	var errs []error

	for _, destination := range m.destinations {
		if err := destination.Submit(ctx, item); err != nil {
			result.Failed[destination.Name()] = err.Error()
			errs = append(errs, fmt.Errorf("%s: %w", destination.Name(), err))
			continue
		}
		result.Succeeded = append(result.Succeeded, destination.Name())
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
