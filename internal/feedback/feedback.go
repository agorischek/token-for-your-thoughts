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
	Feedback  string         `json:"feedback"`
	Source    string         `json:"source"`
	CreatedAt time.Time      `json:"created_at"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

func New(provider, substance, source string, metadata map[string]any) (Item, error) {
	provider = strings.TrimSpace(provider)
	substance = strings.TrimSpace(substance)
	source = strings.TrimSpace(source)

	if provider == "" {
		return Item{}, errors.New("provider is required")
	}
	if substance == "" {
		return Item{}, errors.New("feedback is required")
	}
	if source == "" {
		source = "unknown"
	}

	return Item{
		ID:        uuid.NewString(),
		Provider:  provider,
		Feedback:  substance,
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
	builder.WriteString(i.ID)
	builder.WriteString("\n\n")
	builder.WriteString("- Created At: ")
	builder.WriteString(i.CreatedAt.Format(time.RFC3339))
	builder.WriteString("\n")
	builder.WriteString("- Provider: ")
	builder.WriteString(i.Provider)
	builder.WriteString("\n")
	builder.WriteString("- Source: ")
	builder.WriteString(i.Source)
	builder.WriteString("\n")
	builder.WriteString("- Feedback: ")
	builder.WriteString(i.Feedback)
	builder.WriteString("\n")
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
