// Package server wires the HTTP surface: the dashboard API, the webhook
// intake, the pool endpoints and the SPA.
package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	apierrors "github.com/FacileStudio/tronc/errors"
	"github.com/FacileStudio/tronc/httpjson"
	"github.com/go-chi/chi/v5"

	"github.com/FacileStudio/Antenne/apps/api/internal/app"
	"github.com/FacileStudio/Antenne/apps/api/modules/delivery"
	"github.com/FacileStudio/Antenne/apps/api/modules/events"
	"github.com/FacileStudio/Antenne/apps/api/modules/settings"
)

// maxSettingsBody caps a settings PUT. Generous for a config document, small
// enough that the endpoint cannot be used to exhaust memory.
const maxSettingsBody = 4 << 20

type settingsResponse struct {
	Settings         settings.AppSettings   `json:"settings"`
	Runtime          settings.RuntimeStatus `json:"runtime"`
	MatrixConfigured bool                   `json:"matrixConfigured"`
}

// API serves the dashboard's endpoints.
type API struct {
	app *app.App
}

// NewAPI builds the dashboard API.
func NewAPI(application *app.App) *API { return &API{app: application} }

// RegisterRoutes mounts the dashboard API under the caller's router.
func (a *API) RegisterRoutes(router chi.Router) {
	router.Get("/settings", a.handleGetSettings)
	router.Put("/settings", a.handlePutSettings)
	router.Post("/test-alert", a.handleTestAlert)
	router.Post("/test-delivery-target", a.handleTestTarget)
	router.Post("/test-provider-connection", a.handleTestProvider)
	router.Get("/events", a.handleEvents)
	router.Get("/events/stream", a.handleEventStream)
	router.Get("/events/provider/{providerId}", a.handleProviderEvents)
	router.Get("/events/target/{targetId}", a.handleTargetEvents)
	router.Post("/events/{eventId}/replay", a.handleReplay)
}

func (a *API) handleGetSettings(w http.ResponseWriter, r *http.Request) {
	httpjson.WriteJSON(w, http.StatusOK, a.snapshot())
}

func (a *API) handlePutSettings(w http.ResponseWriter, r *http.Request) {
	raw, err := io.ReadAll(io.LimitReader(r.Body, maxSettingsBody))
	if err != nil {
		httpjson.WriteError(w, apierrors.Invalid("Invalid JSON body."))
		return
	}
	if !json.Valid(raw) {
		httpjson.WriteError(w, apierrors.Invalid("Invalid JSON body."))
		return
	}

	if _, err := a.app.UpdateSettings(r.Context(), raw); err != nil {
		httpjson.WriteError(w, apierrors.Invalid(err.Error()))
		return
	}

	httpjson.WriteJSON(w, http.StatusOK, a.snapshot())
}

func (a *API) snapshot() settingsResponse {
	current := a.app.Settings()
	return settingsResponse{
		Settings:         settings.Redact(current),
		Runtime:          a.app.RuntimeStatus(),
		MatrixConfigured: delivery.HasConfiguredMatrix(current.Delivery),
	}
}

func (a *API) handleTestAlert(w http.ResponseWriter, r *http.Request) {
	if err := a.app.SendTestAlert(r.Context()); err != nil {
		httpjson.WriteError(w, apierrors.Invalid(err.Error()))
		return
	}
	writeOK(w)
}

func (a *API) handleTestTarget(w http.ResponseWriter, r *http.Request) {
	var body struct {
		TargetID string `json:"targetId"`
	}
	if err := httpjson.DecodeJSON(w, r, &body); err != nil || body.TargetID == "" {
		httpjson.WriteError(w, apierrors.Invalid("targetId is required"))
		return
	}
	if err := a.app.TestTarget(r.Context(), body.TargetID); err != nil {
		httpjson.WriteError(w, apierrors.Invalid(err.Error()))
		return
	}
	writeOK(w)
}

func (a *API) handleTestProvider(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ProviderID string `json:"providerId"`
	}
	if err := httpjson.DecodeJSON(w, r, &body); err != nil || body.ProviderID == "" {
		httpjson.WriteError(w, apierrors.Invalid("providerId is required"))
		return
	}
	if err := a.app.TestProvider(body.ProviderID); err != nil {
		httpjson.WriteError(w, apierrors.Invalid(err.Error()))
		return
	}
	writeOK(w)
}

func (a *API) handleEvents(w http.ResponseWriter, r *http.Request) {
	a.writeEventPage(w, events.Query{
		Search: r.URL.Query().Get("q"),
		Source: r.URL.Query().Get("source"),
		Limit:  intParam(r, "limit", 50),
		Offset: intParam(r, "offset", 0),
	})
}

func (a *API) handleProviderEvents(w http.ResponseWriter, r *http.Request) {
	a.writeEventPage(w, events.Query{
		ProviderID: chi.URLParam(r, "providerId"),
		Limit:      intParam(r, "limit", 50),
		Offset:     intParam(r, "offset", 0),
	})
}

func (a *API) handleTargetEvents(w http.ResponseWriter, r *http.Request) {
	a.writeEventPage(w, events.Query{
		TargetID: chi.URLParam(r, "targetId"),
		Limit:    intParam(r, "limit", 50),
		Offset:   intParam(r, "offset", 0),
	})
}

func (a *API) writeEventPage(w http.ResponseWriter, query events.Query) {
	entries, total, err := a.app.Events().List(query)
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}

	// Stats describe the whole filtered set, not the page, so the counters do
	// not change as the operator scrolls.
	stats, err := a.app.Events().StatsFor(events.Query{
		ProviderID: query.ProviderID,
		TargetID:   query.TargetID,
	})
	if err != nil {
		httpjson.WriteError(w, err)
		return
	}

	httpjson.WriteJSON(w, http.StatusOK, map[string]any{
		"events": entries,
		"total":  total,
		"stats":  stats,
	})
}

// handleEventStream is the activity page's live feed. Server-sent events rather
// than a WebSocket: the traffic is one-way and an EventSource reconnects on its
// own.
func (a *API) handleEventStream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		httpjson.WriteError(w, apierrors.Failed("streaming is not supported"))
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	stream, unsubscribe := a.app.Events().Subscribe()
	defer unsubscribe()

	// A quiet Antenne can go minutes without an event, and every proxy in front of
	// one culls an idle connection — Cloudflare at 100 seconds, Traefik at 180.
	// Without this the stream simply dies mid-session and the client reconnects,
	// which a browser's EventSource hides and anything else reports as an error.
	// A comment frame is the SSE keep-alive: every consumer ignores it.
	heartbeat := time.NewTicker(20 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-heartbeat.C:
			if _, err := io.WriteString(w, ": keep-alive\n\n"); err != nil {
				return
			}
			flusher.Flush()
		case entry, open := <-stream:
			if !open {
				return
			}
			encoded, err := json.Marshal(entry)
			if err != nil {
				continue
			}
			fmt.Fprintf(w, "data: %s\n\n", encoded)
			flusher.Flush()
		}
	}
}

func (a *API) handleReplay(w http.ResponseWriter, r *http.Request) {
	var body struct {
		TargetID string `json:"targetId"`
	}
	if err := httpjson.DecodeJSON(w, r, &body); err != nil || body.TargetID == "" {
		httpjson.WriteError(w, apierrors.Invalid("targetId is required"))
		return
	}
	if err := a.app.Replay(r.Context(), chi.URLParam(r, "eventId"), body.TargetID); err != nil {
		httpjson.WriteError(w, apierrors.Invalid(err.Error()))
		return
	}
	writeOK(w)
}

func writeOK(w http.ResponseWriter) {
	httpjson.WriteJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// maxPageSize caps `limit`. Without it one query can ask for the whole log,
// each entry carrying a full envelope.
const maxPageSize = 500

func intParam(r *http.Request, key string, fallback int) int {
	raw := strings.TrimSpace(r.URL.Query().Get(key))
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 0 {
		return fallback
	}
	if key == "limit" && value > maxPageSize {
		return maxPageSize
	}
	return value
}
