package feedback

import (
	"bytes"
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

func (i Item) JSON(pretty bool) ([]byte, error) {
	if pretty {
		return json.MarshalIndent(i, "", "  ")
	}
	return json.Marshal(i)
}

func (i Item) MarkdownEntry() string {
	var builder strings.Builder

	builder.WriteString("## ")
	builder.WriteString(i.ID)
	builder.WriteString("\n\n")
	builder.WriteString(i.Feedback)
	builder.WriteString("\n\n")
	builder.WriteString("_From ")
	builder.WriteString(i.Provider)
	builder.WriteString(" via ")
	builder.WriteString(strings.ToUpper(i.Source))
	builder.WriteString(" at ")
	builder.WriteString(i.CreatedAt.Format(time.RFC3339))
	builder.WriteString("_")
	if len(i.Metadata) > 0 {
		builder.WriteString("\n\n```json\n")
		builder.WriteString(i.MarkdownMetadataJSON())
		builder.WriteString("\n```")
	}

	return builder.String()
}

func (i Item) MarkdownMetadataJSON() string {
	data, err := json.Marshal(i.Metadata)
	if err != nil {
		return "{}"
	}

	var pretty bytes.Buffer
	if err := json.Indent(&pretty, data, "", "  "); err != nil {
		return string(data)
	}
	return pretty.String()
}

func (i Item) String() string {
	return fmt.Sprintf("%s (%s)", i.ID, i.Provider)
}
