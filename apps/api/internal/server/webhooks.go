package server

import (
	"encoding/json"
	"log/slog"
	"net/http"

	apierrors "github.com/FacileStudio/tronc/errors"
	"github.com/FacileStudio/tronc/httpjson"

	"github.com/FacileStudio/Antenne/apps/api/internal/app"
	"github.com/FacileStudio/Antenne/apps/api/internal/envelope"
	"github.com/FacileStudio/Antenne/apps/api/modules/bus"
	"github.com/FacileStudio/Antenne/apps/api/modules/settings"
	"github.com/FacileStudio/Antenne/apps/api/modules/webhooks"
)

// maxWebhookBody caps an incoming webhook. Third-party senders are not trusted
// to be reasonable, and the body is read before authentication because signed
// providers have to hash it.
const maxWebhookBody = 4 << 20

// WebhookHandler receives third-party events.
//
// It is mounted as a catch-all rather than on fixed routes because the paths are
// configurable per provider from the dashboard, and they are already registered
// in third-party systems — a path change must take effect without a restart and
// without touching the router.
type WebhookHandler struct {
	app *app.App
	bus *bus.Server
	log *slog.Logger
}

// NewWebhookHandler builds the intake.
func NewWebhookHandler(application *app.App, busServer *bus.Server, log *slog.Logger) *WebhookHandler {
	return &WebhookHandler{app: application, bus: busServer, log: log}
}

// ServeHTTP dispatches one webhook.
func (h *WebhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	current := h.app.Settings()

	var provider settings.Provider
	found := false
	for _, candidate := range current.Providers.Items {
		if candidate.Enabled && candidate.Type.IsWebhook() && candidate.Path == r.URL.Path {
			provider, found = candidate, true
			break
		}
	}
	if !found {
		httpjson.WriteError(w, apierrors.NotFound("not found"))
		return
	}

	definition, known := webhooks.Lookup(provider.Type)
	if !known {
		httpjson.WriteError(w, apierrors.NotFound("not found"))
		return
	}

	// The body is read once and kept raw: signature verification hashes exactly
	// what the sender signed, so it cannot be re-encoded first.
	body, err := webhooks.ReadBody(r, maxWebhookBody)
	if err != nil {
		httpjson.WriteError(w, apierrors.Invalid("bad request"))
		return
	}

	if !webhooks.Authenticate(definition, provider, r, body) {
		h.log.Warn("webhook rejected",
			slog.String("provider", provider.Name),
			slog.String("path", provider.Path))
		httpjson.WriteError(w, apierrors.Unauthorized(definition.Label+" unauthorized"))
		return
	}

	result, err := definition.Handle(body, webhooks.Context{
		ProviderID:   provider.ID,
		ProviderName: provider.Name,
		ProviderType: provider.Type,
		Path:         provider.Path,
	})
	if err != nil {
		httpjson.WriteError(w, apierrors.Invalid("bad request"))
		return
	}

	h.log.Info("webhook received",
		slog.String("provider", provider.Name),
		slog.String("provider_type", string(provider.Type)),
		slog.String("event", result.Description))

	h.publishToBus(result.EventType, body)

	// A recognised event with nothing worth saying is accepted and not
	// delivered — a heartbeat should not reach Matrix.
	if result.MessageHTML == "" {
		writeOK(w)
		return
	}

	var raw any
	if json.Unmarshal(body, &raw) != nil {
		raw = nil
	}

	_, err = h.app.Dispatch(r.Context(), app.Alert{
		MessageHTML:  result.MessageHTML,
		Source:       envelope.SourceWebhook,
		ProviderID:   provider.ID,
		ProviderName: provider.Name,
		ProviderType: string(provider.Type),
		ProviderTags: provider.Tags,
		Event:        result.Description,
		EventType:    firstNonEmpty(result.EventType, result.Description),
		EventTitle:   firstNonEmpty(result.EventTitle, result.Description),
		EventURL:     result.EventURL,
		OccurredAt:   result.OccurredAt,
		Actor:        result.Actor,
		Subject:      result.Subject,
		Normalized:   result.Normalized,
		RawPayload:   raw,
		// DocuSeal and Ardoise carry personal data in every payload, whatever
		// the individual event says.
		ContainsPII:       result.ContainsPII || provider.Type == settings.ProviderDocuSeal || provider.Type == settings.ProviderArdoise,
		RawPayloadAllowed: result.RawPayloadAllowed,
	})
	if err != nil {
		h.log.Error("webhook alert was not delivered",
			slog.String("provider", provider.Name), slog.Any("error", err))
	}

	writeOK(w)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func (h *WebhookHandler) publishToBus(eventType string, body []byte) {
	if h.bus == nil || eventType == "" {
		return
	}
	if _, err := h.bus.Publish("antenne", eventType, json.RawMessage(body)); err != nil {
		h.log.Warn("webhook event was not published to bus",
			slog.String("event", eventType),
			slog.Any("error", err))
	}
}
