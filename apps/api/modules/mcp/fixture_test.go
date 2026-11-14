package mcp_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/FacileStudio/Antenne/apps/api/internal/authcontext"
	"github.com/FacileStudio/Antenne/apps/api/internal/envelope"
	"github.com/FacileStudio/Antenne/apps/api/internal/server"
	"github.com/FacileStudio/Antenne/apps/api/modules/bus"
	"github.com/FacileStudio/Antenne/apps/api/modules/delivery"
	"github.com/FacileStudio/Antenne/apps/api/modules/events"
	"github.com/FacileStudio/Antenne/apps/api/modules/mcp"
	"github.com/FacileStudio/Antenne/apps/api/modules/settings"
	"github.com/FacileStudio/porte"
	"github.com/FacileStudio/porte/mcpauth"
	"github.com/FacileStudio/tronc/mcp/mcptest"
	"github.com/go-chi/chi/v5"
)

const (
	ownerID     int64 = 1
	secretPool        = "SECRET-POOL-VALUE"
	secretHook        = "SECRET-DISCORD-TOKEN"
	secretToken       = "SECRET-MATRIX-TOKEN"
	secretIMAP        = "SECRET-IMAP-PASSWORD"
)

// identities resolves the owner's subject to the owner.
type identities struct {
	porte.IdentityStore
}

// Find resolves the one known subject.
func (identities) Find(_ context.Context, provider, sub string) (porte.StoredIdentity, error) {
	if sub != "owner" {
		return porte.StoredIdentity{}, porte.ErrNotFound
	}
	return porte.StoredIdentity{UserID: ownerID, Provider: provider, Subject: sub}, nil
}

// accounts knows the owner only.
type accounts struct{}

// IdentityForUser resolves the owner.
func (accounts) IdentityForUser(_ context.Context, userID int64) (authcontext.Identity, error) {
	if userID != ownerID {
		return authcontext.Identity{}, errors.New("unknown")
	}
	return authcontext.Identity{UserID: userID, Email: "owner@example.test", Role: authcontext.RoleAdmin}, nil
}

// pipeline keeps the settings and counts the sends, saving through the real Normalize.
type pipeline struct {
	mu      sync.Mutex
	current settings.AppSettings
	replays map[string]int
	sendErr error
}

func (p *pipeline) Settings() settings.AppSettings {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.current
}

func (p *pipeline) UpdateSettings(_ context.Context, raw json.RawMessage) (settings.AppSettings, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	next, err := settings.Normalize(settings.MergeSecrets(raw, p.current), p.current)
	if err != nil {
		return settings.AppSettings{}, err
	}
	p.current = next
	return next, nil
}

func (p *pipeline) TestTarget(context.Context, string) error { return p.sendErr }

func (p *pipeline) TestProvider(string) error { return nil }

func (p *pipeline) Replay(_ context.Context, eventID, targetID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.replays[eventID+"→"+targetID]++
	return nil
}

// runs counts the replays of one event to one target.
func (p *pipeline) runs(eventID, targetID string) int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.replays[eventID+"→"+targetID]
}

// eventLog holds entries in memory, newest last.
type eventLog struct {
	mu      sync.Mutex
	entries []events.Entry
}

func (l *eventLog) List(q events.Query) ([]events.Entry, int64, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	all := make([]events.Entry, 0, len(l.entries))
	for i := len(l.entries) - 1; i >= 0; i-- {
		all = append(all, l.entries[i])
	}
	end := min(q.Offset+q.Limit, len(all))
	if q.Offset >= len(all) {
		return []events.Entry{}, int64(len(all)), nil
	}
	return all[q.Offset:end], int64(len(all)), nil
}

func (l *eventLog) StatsFor(events.Query) (events.Stats, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return events.Stats{TotalEvents: len(l.entries)}, nil
}

func (l *eventLog) ByID(id string) (events.Entry, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, entry := range l.entries {
		if entry.ID == id {
			return entry, true
		}
	}
	return events.Entry{}, false
}

// add logs one event with a delivery error carrying a webhook URL, and returns its id.
func (l *eventLog) add(title string) string {
	l.mu.Lock()
	defer l.mu.Unlock()
	id := strconv.Itoa(len(l.entries)+1) + "-" + title
	message := "<b>" + title + "</b> happened"
	l.entries = append(l.entries, events.Entry{
		ID: id, Timestamp: time.Now().UTC().Format(time.RFC3339Nano), Source: "webhook", ProviderName: "Sonde",
		ProviderType: "sonde", Event: "monitor down", Message: &message,
		Envelope: &envelope.Envelope{ID: "env-" + id, Event: envelope.Event{Type: "monitor.down", Title: &title}},
		Deliveries: []delivery.Attempt{{
			TargetID: "discord", TargetName: "Discord", Status: "error",
			Error: `Post "https://discord.com/api/webhooks/1/` + secretHook + `": dial tcp: i/o timeout`,
		}},
	})
	return id
}

// health answers a fixed page and probe.
type health struct{}

func (health) HealthPage() server.DeliveryHealthPage {
	return server.DeliveryHealthPage{
		Targets:        []server.TargetHealth{{TargetID: "discord", State: "down", Detail: "https://discord.com/api/webhooks/1/" + secretHook}},
		AnyDeliverable: true,
	}
}

func (health) CheckTarget(_ context.Context, _, targetID string) (server.TargetCheckResult, time.Duration, error) {
	return server.TargetCheckResult{TargetID: targetID, OK: true, Kind: "ok", State: "ok"}, 0, nil
}

// pool is one connected app.
type pool struct{}

func (pool) Stats() bus.Stats {
	return bus.Stats{Connections: 1, Apps: []bus.StatsApp{{AppID: "a1", App: "sonde", Emits: []string{"monitor.down"}}}}
}

// fixture is the MCP endpoint over fakes of Antenne's services, with the confirm page
// routes behind a session the test controls.
type fixture struct {
	issuer   *mcptest.Issuer
	resource string
	pipeline *pipeline
	events   *eventLog
	session  porte.Identity
}

// newFixture starts the endpoint and the confirm page routes on a free port.
func newFixture(t *testing.T) *fixture {
	t.Helper()
	f := &fixture{
		issuer: mcptest.NewIssuer(t), events: &eventLog{},
		pipeline: &pipeline{current: seedSettings(t), replays: map[string]int{}},
		session:  porte.Identity{UserID: ownerID, SessionID: 7, BrowserSession: true},
	}
	var handler http.Handler
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { handler.ServeHTTP(w, r) }))
	t.Cleanup(srv.Close)
	f.resource = srv.URL + "/mcp"
	service := f.service(t)
	router := chi.NewRouter()
	service.RegisterRoutes(router)
	router.Route("/api", func(r chi.Router) {
		r.Use(f.signedIn)
		service.RegisterPendingRoutes(r)
	})
	handler = router
	return f
}

// signedIn attaches the fixture's session, as porte and RequireAuth do.
func (f *fixture) signedIn(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := porte.WithIdentity(r.Context(), f.session)
		ctx = authcontext.With(ctx, authcontext.Identity{UserID: f.session.UserID})
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// service builds the MCP server over the fixture's fakes and issuer.
func (f *fixture) service(t *testing.T) *mcp.Service {
	t.Helper()
	verifier, err := mcpauth.Verifier(context.Background(), mcpauth.Config{
		Issuer: f.issuer.URL, Resource: f.resource, ScopePrefix: "antenne", Identities: identities{},
	})
	must(t, err)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	service, err := mcp.NewService(mcp.Deps{
		Pipeline: f.pipeline, Events: f.events, Health: health{}, Pool: pool{}, Accounts: accounts{},
	}, mcp.Config{
		Resource: f.resource, Issuer: f.issuer.URL, Version: "test", StateKey: []byte("state-key"),
		Verifier: verifier, Pending: mcptest.NewPending(), PerMinute: 100_000, Logger: logger,
	})
	must(t, err)
	return service
}

// token mints the owner's token with scopes, both by default.
func (f *fixture) token(t testing.TB, scopes ...string) string {
	if len(scopes) == 0 {
		scopes = []string{"antenne.read", "antenne.write"}
	}
	return f.issuer.Token(t, f.resource, "owner", scopes...)
}

// seedSettings is a configuration holding a credential in every place one can sit.
func seedSettings(t *testing.T) settings.AppSettings {
	t.Helper()
	password := secretIMAP
	doc := settings.AppSettings{
		Timeout: 10,
		Bus:     &settings.BusSettings{Secret: secretPool},
		Providers: settings.ProvidersConfig{Port: 9090, Items: []settings.Provider{
			{ID: "sonde", Type: settings.ProviderSonde, Name: "Sonde", Enabled: true, Tags: []string{"ops"},
				WebhookSettings: settings.WebhookSettings{Path: "/sonde"}},
			{ID: "inbox", Type: settings.ProviderIMAP, Name: "Inbox", Enabled: true,
				IMAPSettings: settings.IMAPSettings{Host: "imap.example.test", Port: 993, Username: "me", Password: &password}},
		}},
		Delivery: settings.DeliveryConfig{Items: []settings.DeliveryTarget{
			{ID: "discord", Type: settings.TargetDiscord, Name: "Discord", Enabled: true, ProviderIDs: []string{"sonde"},
				DiscordSettings: settings.DiscordSettings{WebhookURL: "https://discord.com/api/webhooks/1/" + secretHook}},
			{ID: "matrix", Type: settings.TargetMatrix, Name: "Matrix", Enabled: true, AccessToken: secretToken,
				MatrixSettings: settings.MatrixSettings{Homeserver: "https://matrix.example.test", RoomIDs: []string{"!r:example.test"}}},
		}},
	}
	raw, err := json.Marshal(doc)
	must(t, err)
	normalized, err := settings.Normalize(raw, settings.AppSettings{})
	must(t, err)
	return normalized
}

// must fails the test on err.
func must(t testing.TB, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
