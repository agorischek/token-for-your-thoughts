package feedback

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Item struct {
	ID        string         `json:"id"`
	Provider  string         `json:"provider"`
	Summary   string         `json:"summary"`
	Details   string         `json:"details,omitempty"`
	Category  string         `json:"category,omitempty"`
	Source    string         `json:"source"`
	CreatedAt time.Time      `json:"created_at"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

func New(provider, summary, details, category, source string, metadata map[string]any) (Item, error) {
	provider = strings.TrimSpace(provider)
	summary = strings.TrimSpace(summary)
	details = strings.TrimSpace(details)
	category = strings.TrimSpace(category)
	source = strings.TrimSpace(source)

	if provider == "" {
		return Item{}, errors.New("provider is required")
	}
	if summary == "" {
		return Item{}, errors.New("summary is required")
	}
	if source == "" {
		source = "unknown"
	}

	return Item{
		ID:        uuid.NewString(),
		Provider:  provider,
		Summary:   summary,
		Details:   details,
		Category:  category,
		Source:    source,
		CreatedAt: time.Now().UTC(),
		Metadata:  metadata,
	}, nil
}

func (i Item) MetadataJSON() string {
	if len(i.Metadata) == 0 {
		return "{}"
	}

	data, err := json.Marshal(i.Metadata)
	if err != nil {
		return "{}"
	}
	return string(data)
}

func (i Item) MarkdownEntry() string {
	var builder strings.Builder

	builder.WriteString("## ")
	builder.WriteString(i.CreatedAt.Format(time.RFC3339))
	builder.WriteString("\n\n")
	builder.WriteString("- ID: `")
	builder.WriteString(i.ID)
	builder.WriteString("`\n")
	builder.WriteString("- Provider: ")
	builder.WriteString(i.Provider)
	builder.WriteString("\n")
	if i.Category != "" {
		builder.WriteString("- Category: ")
		builder.WriteString(i.Category)
		builder.WriteString("\n")
	}
	builder.WriteString("- Source: ")
	builder.WriteString(i.Source)
	builder.WriteString("\n")
	builder.WriteString("- Summary: ")
	builder.WriteString(i.Summary)
	builder.WriteString("\n")
	if i.Details != "" {
		builder.WriteString("\n")
		builder.WriteString(i.Details)
		builder.WriteString("\n")
	}
	if len(i.Metadata) > 0 {
		builder.WriteString("\n```json\n")
		builder.WriteString(i.MetadataJSON())
		builder.WriteString("\n```\n")
	}
	builder.WriteString("\n")

	return builder.String()
}

func (i Item) String() string {
	return fmt.Sprintf("%s (%s)", i.ID, i.Provider)
}
