package server_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/FacileStudio/Antenne/apps/api/internal/app"
	"github.com/FacileStudio/Antenne/apps/api/internal/pgtest"
	"github.com/FacileStudio/Antenne/apps/api/internal/server"
	"github.com/FacileStudio/Antenne/apps/api/modules/delivery"
	"github.com/FacileStudio/Antenne/apps/api/modules/events"
	"github.com/FacileStudio/Antenne/apps/api/modules/health"
	"github.com/FacileStudio/Antenne/apps/api/modules/settings"
)

const webhookToken = "SECRET-WEBHOOK-TOKEN-0123456789"

// TestNoDashboardResponseCarriesAStoredWebhookURL seeds the rows Antenne wrote
// before v1.6.0, a delivery error and a health detail quoting a Discord webhook URL
// whose path is its credential, plus a target that fails to send, and checks that
// no dashboard response carries the token while the host still shows.
func TestNoDashboardResponseCarriesAStoredWebhookURL(t *testing.T) {
	db := pgtest.New(t)
	log := slog.New(slog.DiscardHandler)
	webhook := "http://127.0.0.1:1/api/webhooks/42/" + webhookToken
	stored := "Post \"https://discord.com/api/webhooks/42/" + webhookToken + "?wait=true\": dial tcp: i/o timeout"

	store, err := settings.NewStore(db, log, settings.Defaults(9090, 2))
	if err != nil {
		t.Fatal(err)
	}
	config := store.Get()
	config.Delivery.Items = []settings.DeliveryTarget{{
		ID: "d1", Type: settings.TargetDiscord, Name: "Discord", Enabled: true,
		DiscordSettings: settings.DiscordSettings{WebhookURL: webhook},
	}}
	raw, err := json.Marshal(config)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.Update(raw); err != nil {
		t.Fatal(err)
	}

	now := time.Now()
	records := health.NewStore(db)
	if err := records.Upsert(context.Background(), health.Record{
		TargetID: "d1", State: health.StateDown, DownSince: &now,
		LastSeen: health.LastSeen{LastObservedAt: &now, LastFailureAt: &now, LastDetail: stored},
	}); err != nil {
		t.Fatal(err)
	}
	deliveries, _ := json.Marshal([]delivery.Attempt{{TargetID: "d1", TargetName: "Discord", TargetType: "discord", Status: delivery.StatusError, Error: stored}})
	envelopeJSON := `{"event":{"type":"test"},"subject":{"type":"system"},"payload":{}}`
	if err := db.Exec(`INSERT INTO events (id, occurred_at, source, provider_id, event, envelope, deliveries, search_text)
		VALUES ('e1', now(), 'test', 'p1', 'old', ?::jsonb, ?::jsonb, '')`, envelopeJSON, string(deliveries)).Error; err != nil {
		t.Fatal(err)
	}

	sender := delivery.NewSender(log)
	application := app.New(log, store, events.NewService(db, log), sender, settings.RuntimeStatus{})
	application.AttachHealth(app.HealthDeps{Records: records, Probe: sender.Probe, Policy: health.Evaluate})
	router := chi.NewRouter()
	server.NewAPI(application).RegisterRoutes(router)

	requests := []struct{ method, path, body string }{
		{http.MethodGet, "/settings", ""},
		{http.MethodGet, "/events", ""},
		{http.MethodGet, "/events?q=discord", ""},
		{http.MethodGet, "/events/target/d1", ""},
		{http.MethodGet, "/events/provider/p1", ""},
		{http.MethodGet, "/delivery-health", ""},
		{http.MethodPost, "/delivery-health/d1/check", ""},
		{http.MethodPost, "/test-delivery-target", `{"targetId":"d1"}`},
		{http.MethodPost, "/events/e1/replay", `{"targetId":"d1"}`},
		{http.MethodPost, "/test-alert", ""},
	}
	sawHost := false
	for _, r := range requests {
		response := send(router, r.method, r.path, r.body, map[string]string{"Content-Type": "application/json"})
		body := response.Body.String()
		if strings.Contains(body, webhookToken) || strings.Contains(body, "/api/webhooks/") {
			t.Errorf("%s %s returned the webhook credential: %s", r.method, r.path, body)
		}
		sawHost = sawHost || strings.Contains(body, "https://discord.com")
	}
	if !sawHost {
		t.Error("no response kept the host, so the redaction removed the diagnostic too")
	}
}
