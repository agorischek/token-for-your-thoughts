package destinations

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/agorischek/token-for-your-thoughts/internal/config"
	"github.com/agorischek/token-for-your-thoughts/internal/feedback"
)

const defaultApplicationInsightsIngestionEndpoint = "https://dc.services.visualstudio.com/"

type ApplicationInsightsDestination struct {
	client             *http.Client
	ingestionEndpoint  string
	instrumentationKey string
	eventName          string
}

type applicationInsightsEnvelope struct {
	Name string                         `json:"name"`
	Time string                         `json:"time"`
	IKey string                         `json:"iKey"`
	Tags map[string]string              `json:"tags,omitempty"`
	Data applicationInsightsDataWrapper `json:"data"`
}

type applicationInsightsDataWrapper struct {
	BaseType string                           `json:"baseType"`
	BaseData applicationInsightsEventBaseData `json:"baseData"`
}

type applicationInsightsEventBaseData struct {
	Ver        int               `json:"ver"`
	Name       string            `json:"name"`
	Properties map[string]string `json:"properties"`
}

type applicationInsightsTrackResponse struct {
	ItemsReceived int `json:"itemsReceived"`
	ItemsAccepted int `json:"itemsAccepted"`
	Errors        []struct {
		Index      int    `json:"index"`
		StatusCode int    `json:"statusCode"`
		Message    string `json:"message"`
	} `json:"errors"`
}

func NewApplicationInsightsDestination(cfg config.DestinationConfig) (*ApplicationInsightsDestination, error) {
	return newApplicationInsightsDestination(cfg, &http.Client{Timeout: 10 * time.Second})
}

func newApplicationInsightsDestination(cfg config.DestinationConfig, client *http.Client) (*ApplicationInsightsDestination, error) {
	parts, err := parseConnectionString(cfg.ConnectionString)
	if err != nil {
		return nil, err
	}

	ikey := strings.TrimSpace(parts["InstrumentationKey"])
	if ikey == "" {
		return nil, fmt.Errorf("application_insights connection string requires InstrumentationKey")
	}

	endpoint := strings.TrimSpace(parts["IngestionEndpoint"])
	if endpoint == "" {
		if suffix := strings.TrimSpace(parts["EndpointSuffix"]); suffix != "" {
			endpoint = "https://dc." + suffix
		} else {
			endpoint = defaultApplicationInsightsIngestionEndpoint
		}
	}
	if !strings.HasSuffix(endpoint, "/") {
		endpoint += "/"
	}

	return &ApplicationInsightsDestination{
		client:             client,
		ingestionEndpoint:  endpoint,
		instrumentationKey: ikey,
		eventName:          cfg.EventName,
	}, nil
}

func (s *ApplicationInsightsDestination) Name() string {
	return "application_insights"
}

func (s *ApplicationInsightsDestination) Submit(ctx context.Context, item feedback.Item) error {
	properties := map[string]string{
		"feedback.id":            item.ID,
		"feedback.provider":      item.Provider,
		"feedback.source":        item.Source,
		"feedback.text":          item.Feedback,
		"feedback.created_at":    item.CreatedAt.Format(time.RFC3339Nano),
		"feedback.metadata_json": item.MetadataJSON(),
	}

	payload := []applicationInsightsEnvelope{{
		Name: "Microsoft.ApplicationInsights.Event",
		Time: item.CreatedAt.Format(time.RFC3339Nano),
		IKey: s.instrumentationKey,
		Tags: map[string]string{
			"ai.cloud.role": "tfyt",
		},
		Data: applicationInsightsDataWrapper{
			BaseType: "EventData",
			BaseData: applicationInsightsEventBaseData{
				Ver:        2,
				Name:       s.eventName,
				Properties: properties,
			},
		},
	}}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal application insights payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.ingestionEndpoint+"v2/track", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create application insights request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("send application insights telemetry: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read application insights response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("application insights ingestion returned %s: %s", resp.Status, strings.TrimSpace(string(respBody)))
	}

	var trackResp applicationInsightsTrackResponse
	if len(respBody) > 0 && json.Unmarshal(respBody, &trackResp) == nil {
		if trackResp.ItemsAccepted < trackResp.ItemsReceived {
			if len(trackResp.Errors) > 0 {
				return fmt.Errorf("application insights accepted %d of %d items: %s", trackResp.ItemsAccepted, trackResp.ItemsReceived, trackResp.Errors[0].Message)
			}
			return fmt.Errorf("application insights accepted %d of %d items", trackResp.ItemsAccepted, trackResp.ItemsReceived)
		}
	}

	return nil
}

func parseConnectionString(raw string) (map[string]string, error) {
	parts := map[string]string{}
	for _, part := range strings.Split(raw, ";") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		key, value, found := strings.Cut(part, "=")
		if !found {
			return nil, fmt.Errorf("invalid connection string segment %q", part)
		}
		parts[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}
	if len(parts) == 0 {
		return nil, fmt.Errorf("application_insights connection string is empty")
	}
	return parts, nil
}
