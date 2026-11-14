// Package app is Antenne's alert pipeline: the one path every event takes from a
// provider to the delivery targets, and the runtime that keeps the pollers in
// step with the settings.
package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/FacileStudio/Antenne/apps/api/internal/envelope"
	"github.com/FacileStudio/Antenne/apps/api/modules/delivery"
	"github.com/FacileStudio/Antenne/apps/api/modules/events"
	"github.com/FacileStudio/Antenne/apps/api/modules/providers"
	"github.com/FacileStudio/Antenne/apps/api/modules/settings"
	"github.com/FacileStudio/tronc/joblock"
)

// dedupWindow is how long an identical message from the same provider is
// swallowed. Two sources reporting one outage within half a minute is one
// alert, not two.
const dedupWindow = 30 * time.Second

// Alert is one event entering the pipeline.
type Alert struct {
	MessageHTML       string
	Source            envelope.Source
	ProviderID        string
	ProviderName      string
	ProviderType      string
	ProviderTags      []string
	Event             string
	EventType         string
	EventTitle        string
	EventURL          string
	OccurredAt        string
	Actor             *envelope.Actor
	Subject           *envelope.Subject
	Normalized        map[string]any
	RawPayload        any
	ContainsPII       bool
	RawPayloadAllowed bool
}

// App is the running service.
type App struct {
	log      *slog.Logger
	store    *settings.Store
	events   *events.Service
	sender   *delivery.Sender
	retries  *delivery.RetryQueue
	monitor  *providers.Monitor
	imap     *providers.IMAP
	rss      *providers.RSS
	health   *healthRuntime
	status   settings.RuntimeStatus
	statusMu sync.RWMutex

	// pollCtx is the lifetime of this replica's hold on the pollers lock, set
	// while it holds it and nil otherwise. The pollers are bound to it and never
	// to a request context: a save that re-applied the runtime under
	// r.Context() would start every watcher and have it cancelled the moment the
	// HTTP response was written, leaving monitoring silently off until the next
	// restart.
	pollMu  sync.Mutex
	pollCtx context.Context

	dedupMu sync.Mutex
	dedup   map[string]time.Time

	// onSettingsApplied lets the pool server pick up a secret rotated from the
	// dashboard without a restart. It is a callback rather than a direct
	// reference because the pool depends on the app for its activity log, and
	// the reverse edge would close the cycle.
	onSettingsApplied func(settings.AppSettings)

	// notifyingFailure stops a failing delivery target from generating a
	// delivery-failure alert that fails, that generates another alert.
	notifyingFailure sync.Mutex
	inFailureNotice  bool
}

// New builds the pipeline.
func New(log *slog.Logger, store *settings.Store, eventLog *events.Service, sender *delivery.Sender, status settings.RuntimeStatus) *App {
	application := &App{
		log:    log,
		store:  store,
		events: eventLog,
		sender: sender,
		status: status,
		dedup:  map[string]time.Time{},
	}

	application.retries = delivery.NewRetryQueue(sender, log, eventLog.MarkRetry)
	application.retries.OnExhausted(application.notifyDeliveryExhausted)
	application.monitor = providers.NewMonitor(log, application.emitFromProvider(envelope.SourceWebsite))
	application.imap = providers.NewIMAP(log, application.emitFromProvider(envelope.SourceIMAP))
	application.rss = providers.NewRSS(log, application.emitFromProvider(envelope.SourceRSS))

	return application
}

// Settings returns the live configuration.
func (a *App) Settings() settings.AppSettings { return a.store.Get() }

// Store exposes the settings store to the HTTP layer.
func (a *App) Store() *settings.Store { return a.store }

// Events exposes the activity log to the HTTP layer.
func (a *App) Events() *events.Service { return a.events }

// RuntimeStatus is what the dashboard shows about the process.
func (a *App) RuntimeStatus() settings.RuntimeStatus {
	a.statusMu.RLock()
	defer a.statusMu.RUnlock()
	status := a.status
	status.UsingPersistedSettings = a.store.Persisted()
	return status
}

// pollersLockKey names the Postgres advisory lock that elects the replica
// running the website, IMAP, RSS and health pollers. The retry queue is not
// behind it: it is in memory, so each replica retries what it enqueued.
const pollersLockKey = "antenne-pollers"

// Start starts the retry queue and the heartbeat, and runs the pollers on the
// replica that holds the pollers lock in db. The context it takes is the
// process lifetime; everything long-running hangs off it.
func (a *App) Start(ctx context.Context, db *sql.DB) {
	a.retries.Start(ctx)
	a.health.Start(ctx)
	go func() {
		_ = joblock.Hold(ctx, db, pollersLockKey, a.runPollers, joblock.WithOnError(func(err error) {
			a.log.Warn("pollers lock", slog.Any("error", err))
		}))
	}()
}

func (a *App) runPollers(held context.Context) {
	a.pollMu.Lock()
	a.pollCtx = held
	a.pollMu.Unlock()
	a.log.Info("this replica now runs the pollers")
	a.applyRuntime(a.store.Get())
	<-held.Done()
	a.pollMu.Lock()
	a.pollCtx = nil
	a.pollMu.Unlock()
	a.stopPollers()
	a.log.Info("this replica stopped the pollers")
}

func (a *App) stopPollers() {
	a.monitor.Stop()
	a.imap.Stop()
	a.rss.Stop()
	a.health.StopMonitor()
}

// Stop halts the pollers.
func (a *App) Stop() {
	a.stopPollers()
	a.health.Stop()
}

// OnSettingsApplied registers a callback fired whenever the runtime is
// re-applied, on boot and after every dashboard save.
func (a *App) OnSettingsApplied(callback func(settings.AppSettings)) {
	a.onSettingsApplied = callback
}

func (a *App) applyRuntime(current settings.AppSettings) {
	a.pollMu.Lock()
	ctx := a.pollCtx
	a.pollMu.Unlock()
	if ctx != nil {
		a.monitor.Apply(ctx, current)
		a.imap.Apply(ctx, current)
		a.rss.Apply(ctx, current)
		a.health.Apply(ctx, current)
	}

	if a.onSettingsApplied != nil {
		a.onSettingsApplied(current)
	}

	a.statusMu.Lock()
	a.status.LastReloadedAt = time.Now().UTC().Format(time.RFC3339)
	a.statusMu.Unlock()
}

// UpdateSettings persists a dashboard save, re-applies the runtime and fires
// whichever system events the operator enabled.
func (a *App) UpdateSettings(ctx context.Context, raw json.RawMessage) (settings.AppSettings, error) {
	merged := settings.MergeSecrets(raw, a.store.Get())

	previous, next, err := a.store.Update(merged)
	if err != nil {
		return settings.AppSettings{}, err
	}

	a.applyRuntime(next)
	a.log.Info("settings updated",
		slog.Int("providers", len(next.Providers.Items)),
		slog.Int("delivery_targets", len(next.Delivery.Items)))

	a.fireProviderEvents(ctx, previous, next)
	return next, nil
}

// AnnounceStartup sends the boot notification, when it is enabled and anything
// could receive it.
func (a *App) AnnounceStartup(ctx context.Context) {
	current := a.store.Get()

	if !delivery.HasAny(current.Delivery) {
		a.log.Warn("no delivery target is configured yet; the dashboard will still start")
		return
	}
	if !current.System.Events.Startup {
		return
	}

	attempts, err := a.Dispatch(ctx, Alert{
		MessageHTML:  "<b>Antenne started</b><br/>Dashboard and API ready.",
		Source:       envelope.SourceSystem,
		ProviderID:   settings.SystemProviderID,
		ProviderName: "Antenne",
		ProviderType: "system",
		Event:        "antenne started",
		EventType:    "system.startup",
		Subject:      &envelope.Subject{Type: envelope.SubjectSystem, Name: envelope.Str("Antenne")},
	})
	switch {
	case err != nil:
		a.log.Error("startup notification failed", slog.Any("error", err))
	case len(attempts) == 0:
		a.log.Info("startup notification skipped: no target selected the system provider")
	default:
		a.log.Info("startup notification delivered", slog.Int("targets", len(attempts)))
	}
}

// SendTestAlert delivers a test through the whole pipeline.
func (a *App) SendTestAlert(ctx context.Context) error {
	current := a.store.Get()
	label := "dashboard"
	if len(current.Providers.Items) > 0 {
		label = current.Providers.Items[0].Name
	}

	_, err := a.Dispatch(ctx, Alert{
		MessageHTML: fmt.Sprintf("🧪 <b>Antenne test alert</b><br/>Dashboard config for <b>%s</b> is live.",
			envelope.EscapeHTML(label)),
		Source:       envelope.SourceTest,
		ProviderID:   settings.SystemProviderID,
		ProviderName: "Antenne",
		ProviderType: "system",
		Event:        "test alert",
		EventType:    "system.test",
		Subject:      &envelope.Subject{Type: envelope.SubjectSystem, Name: envelope.Str("Antenne")},
	})
	return err
}

// TestTarget sends one message straight to one target, bypassing the routing
// rules — the dashboard's "does this target work at all" button.
func (a *App) TestTarget(ctx context.Context, targetID string) error {
	current := a.store.Get()
	target, ok := current.FindTarget(targetID)
	if !ok {
		return errors.New("delivery target not found")
	}

	message := fmt.Sprintf("🧪 <b>Antenne target test</b><br/>Target <b>%s</b> is configured.",
		envelope.EscapeHTML(target.Name))

	return a.sender.SendTo(ctx, target, delivery.Message{
		HTML: message,
		Envelope: envelope.Build(envelope.Input{
			Source:       envelope.SourceTest,
			ProviderID:   envelope.Str(settings.SystemProviderID),
			ProviderType: "system",
			ProviderName: "Antenne",
			EventType:    "target test",
			EventTitle:   "Antenne target test",
			MessageHTML:  message,
			Subject: &envelope.Subject{
				Type: envelope.SubjectSystem,
				ID:   envelope.Str(target.ID),
				Name: envelope.Str(target.Name),
			},
			NormalizedPayload: map[string]any{
				"targetId":   target.ID,
				"targetType": string(target.Type),
			},
		}),
	})
}

// TestProvider verifies a provider's connection. Only IMAP has one to test.
func (a *App) TestProvider(providerID string) error {
	provider, ok := a.store.Get().FindProvider(providerID)
	if !ok {
		return errors.New("provider not found")
	}
	if provider.Type != settings.ProviderIMAP {
		return errors.New("connection testing is only supported for IMAP providers")
	}
	return providers.TestConnection(provider)
}

// Replay re-sends a logged event to one target.
func (a *App) Replay(ctx context.Context, eventID, targetID string) error {
	entry, ok := a.events.ByID(eventID)
	if !ok || entry.Envelope == nil {
		return errors.New("event not found or has no envelope")
	}

	target, ok := a.store.Get().FindTarget(targetID)
	if !ok {
		return errors.New("delivery target not found")
	}

	message := ""
	if entry.Message != nil {
		message = *entry.Message
	}

	return a.sender.SendTo(ctx, target, delivery.Message{HTML: message, Envelope: *entry.Envelope})
}

// LogBusActivity records a bus lifecycle event in the activity feed.
func (a *App) LogBusActivity(eventType, app, detail string) {
	if !a.store.Get().Bus.LogActivityEnabled() {
		return
	}
	a.events.Push(events.New{
		Source:       string(envelope.SourceSystem),
		ProviderName: app,
		ProviderType: "bus",
		Event:        eventType,
		Message:      detail,
	})
}
