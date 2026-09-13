package webhooks

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/FacileStudio/Antenne/apps/api/modules/settings"
)

// genericPayload is the loose shape any app can send to /webhook/generic.
// Every field is optional; data carries whatever else the sender wants kept.
type genericPayload struct {
	Title   string         `json:"title"`
	Message string         `json:"message"`
	Event   string         `json:"event"`
	URL     string         `json:"url"`
	Source  string         `json:"source"`
	Data    map[string]any `json:"data"`
}

// generic is the passthrough provider: a sender that does not deserve its own
// payload dialect posts what it has and Antenne renders it verbatim. The
// sha256= body signature follows the suite's convention, so a sender that
// already talks to a signed provider needs no new signing code.
var generic = Definition{
	Type:         settings.ProviderGeneric,
	Label:        "Generic",
	Authenticate: signedWith("sha256="),
	Handle: func(body []byte, _ Context) (Result, error) {
		var payload genericPayload
		if err := decode(body, &payload); err != nil {
			return genericPlain(body)
		}
		return genericResult(payload), nil
	},
}

func genericResult(payload genericPayload) Result {
	return Result{
		Description: firstNonEmpty(payload.Source, "generic") + " event",
		MessageHTML: genericMessage(payload),
		EventType:   firstNonEmpty(payload.Event, "generic.event"),
		EventTitle:  firstNonEmpty(payload.Title, payload.Event, payload.Source),
		EventURL:    payload.URL,
		Normalized:  genericNormalized(payload),
	}
}

func genericMessage(payload genericPayload) string {
	parts := []string{genericHeading(payload)}
	if payload.Message != "" {
		parts = append(parts, strings.ReplaceAll(escape(payload.Message), "\n", "<br/>"))
	}
	parts = append(parts, genericDataLines(payload.Data)...)
	return join(parts)
}

func genericHeading(payload genericPayload) string {
	title := firstNonEmpty(payload.Title, payload.Event)
	switch {
	case title == "" && payload.Source == "":
		return ""
	case title == "":
		return "<b>[" + escape(payload.Source) + "]</b>"
	case payload.Source == "":
		return "<b>" + escape(title) + "</b>"
	default:
		return "<b>[" + escape(payload.Source) + "] " + escape(title) + "</b>"
	}
}

func genericDataLines(data map[string]any) []string {
	if len(data) == 0 {
		return nil
	}

	keys := make([]string, 0, len(data))
	for key := range data {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	lines := make([]string, 0, len(data))
	for _, key := range keys {
		lines = append(lines, escape(key)+": "+escape(fmt.Sprintf("%v", data[key])))
	}
	return lines
}

func genericNormalized(payload genericPayload) map[string]any {
	normalized := map[string]any{
		"title":   payload.Title,
		"message": payload.Message,
		"event":   firstNonEmpty(payload.Event, "generic.event"),
		"url":     payload.URL,
		"source":  firstNonEmpty(payload.Source, "generic"),
	}
	for key, value := range payload.Data {
		normalized[key] = value
	}
	return normalized
}

func genericPlain(body []byte) (Result, error) {
	var raw any
	if err := json.Unmarshal(body, &raw); err != nil {
		return Result{}, err
	}
	rendered, err := json.Marshal(raw)
	if err != nil {
		return Result{}, err
	}
	return Result{
		Description: "generic event",
		MessageHTML: escape(string(rendered)),
		EventType:   "generic.event",
		Normalized:  map[string]any{"body": raw},
	}, nil
}
