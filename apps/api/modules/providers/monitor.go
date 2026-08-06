// Package providers drives the sources Nook polls itself: websites, IMAP
// inboxes and RSS feeds. Webhook providers are handled by the webhooks package;
// these three are the ones Nook has to go and ask.
package providers

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/FacileStudio/Nook/apps/api/internal/envelope"
	"github.com/FacileStudio/Nook/apps/api/modules/settings"
)

// Alert is what a poller emits. The manager itself never delivers anything; the
// app wires this to the delivery pipeline.
type Alert struct {
	MessageHTML  string
	ProviderID   string
	ProviderName string
	ProviderType string
	Event        string
	EventType    string
	EventTitle   string
	EventURL     string
	OccurredAt   string
	Actor        *envelope.Actor
	Subject      *envelope.Subject
	Normalized   map[string]any
	ContainsPII  bool
}

// Emit is the callback a manager sends alerts through.
type Emit func(Alert)

type siteState struct {
	up                  bool
	lastChange          time.Time
	consecutiveFailures int
}

// Monitor watches website providers and reports up and down transitions.
type Monitor struct {
	log    *slog.Logger
	emit   Emit
	client *http.Client

	mu     sync.Mutex
	states map[string]*siteState
	cancel context.CancelFunc
	group  sync.WaitGroup
}

// NewMonitor builds the website monitor.
func NewMonitor(log *slog.Logger, emit Emit) *Monitor {
	return &Monitor{
		log:    log,
		emit:   emit,
		client: &http.Client{},
		states: map[string]*siteState{},
	}
}

// Apply restarts the watchers to match the current settings. It is called on
// boot and after every dashboard save.
func (m *Monitor) Apply(ctx context.Context, current settings.AppSettings) {
	m.Stop()

	loopCtx, cancel := context.WithCancel(ctx)
	m.mu.Lock()
	m.cancel = cancel
	m.states = map[string]*siteState{}
	m.mu.Unlock()

	for _, provider := range current.Providers.Items {
		if provider.Type != settings.ProviderWebsite || !provider.Enabled {
			continue
		}

		m.mu.Lock()
		m.states[provider.ID] = &siteState{up: true, lastChange: time.Now()}
		m.mu.Unlock()

		m.log.Info("monitoring website",
			slog.String("provider", provider.Name),
			slog.String("url", provider.URL),
			slog.Int("interval_seconds", provider.Interval))

		m.group.Add(1)
		go m.watch(loopCtx, provider, time.Duration(current.Timeout)*time.Second)
	}
}

// Stop halts every watcher and waits for the in-flight checks to finish.
func (m *Monitor) Stop() {
	m.mu.Lock()
	cancel := m.cancel
	m.cancel = nil
	m.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	m.group.Wait()
}

func (m *Monitor) watch(ctx context.Context, provider settings.Provider, timeout time.Duration) {
	defer m.group.Done()

	interval := time.Duration(provider.Interval) * time.Second
	if interval <= 0 {
		interval = time.Minute
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	m.check(ctx, provider, timeout)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.check(ctx, provider, timeout)
		}
	}
}

func (m *Monitor) check(ctx context.Context, provider settings.Provider, timeout time.Duration) {
	requestCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	request, err := http.NewRequestWithContext(requestCtx, http.MethodGet, provider.URL, nil)
	if err != nil {
		m.fail(provider, err.Error())
		return
	}
	request.Header.Set("User-Agent", "Nook/1.0 uptime-bot")

	response, err := m.client.Do(request)
	if err != nil {
		m.log.Warn("website check failed",
			slog.String("provider", provider.Name),
			slog.String("url", provider.URL),
			slog.Any("error", err))
		m.fail(provider, err.Error())
		return
	}
	response.Body.Close()

	m.log.Debug("website checked",
		slog.String("provider", provider.Name),
		slog.String("url", provider.URL),
		slog.Int("status", response.StatusCode))

	// Anything under 400 counts as healthy: several monitored hosts redirect
	// their canonical path or locale, and treating a 3xx as downtime would page
	// on a working site.
	if response.StatusCode >= 400 {
		m.fail(provider, fmt.Sprintf("HTTP %d", response.StatusCode))
		return
	}
	m.succeed(provider)
}

func (m *Monitor) fail(provider settings.Provider, reason string) {
	m.mu.Lock()
	state, ok := m.states[provider.ID]
	if !ok {
		m.mu.Unlock()
		return
	}

	state.consecutiveFailures++
	// Two consecutive failures before alerting: one timeout is noise, two in a
	// row is an outage.
	shouldAlert := state.up && state.consecutiveFailures >= 2
	if shouldAlert {
		state.up = false
		state.lastChange = time.Now()
	}
	failures := state.consecutiveFailures
	m.mu.Unlock()

	if !shouldAlert {
		return
	}

	m.emit(Alert{
		MessageHTML: fmt.Sprintf("<b>[Website] %s</b> is DOWN — %s<br/><code>%s</code>",
			envelope.EscapeHTML(provider.Name), envelope.EscapeHTML(reason), envelope.EscapeHTML(provider.URL)),
		ProviderID:   provider.ID,
		ProviderName: provider.Name,
		ProviderType: string(settings.ProviderWebsite),
		Event:        "site DOWN",
		EventType:    "website.down",
		EventTitle:   provider.Name + " is down",
		Subject: &envelope.Subject{
			Type: envelope.SubjectService,
			ID:   envelope.Str(provider.ID),
			Name: envelope.Str(provider.Name),
			URL:  envelope.Str(provider.URL),
		},
		Normalized: map[string]any{
			"url":                 provider.URL,
			"status":              "down",
			"reason":              reason,
			"consecutiveFailures": failures,
		},
	})
}

func (m *Monitor) succeed(provider settings.Provider) {
	m.mu.Lock()
	state, ok := m.states[provider.ID]
	if !ok {
		m.mu.Unlock()
		return
	}

	recovered := !state.up
	downtime := time.Since(state.lastChange)
	state.consecutiveFailures = 0
	if recovered {
		state.up = true
		state.lastChange = time.Now()
	}
	m.mu.Unlock()

	if !recovered {
		return
	}

	humanized := humanDuration(downtime)
	m.emit(Alert{
		MessageHTML: fmt.Sprintf("<b>[Website] %s</b> is back up — was down for %s",
			envelope.EscapeHTML(provider.Name), humanized),
		ProviderID:   provider.ID,
		ProviderName: provider.Name,
		ProviderType: string(settings.ProviderWebsite),
		Event:        "site UP",
		EventType:    "website.up",
		EventTitle:   provider.Name + " is back up",
		Subject: &envelope.Subject{
			Type: envelope.SubjectService,
			ID:   envelope.Str(provider.ID),
			Name: envelope.Str(provider.Name),
			URL:  envelope.Str(provider.URL),
		},
		Normalized: map[string]any{
			"url":      provider.URL,
			"status":   "up",
			"downtime": humanized,
		},
	})
}

func humanDuration(span time.Duration) string {
	switch {
	case span < time.Minute:
		return fmt.Sprintf("%ds", int(span.Seconds()))
	case span < time.Hour:
		return fmt.Sprintf("%dm", int(span.Minutes()))
	case span < 24*time.Hour:
		return fmt.Sprintf("%dh %dm", int(span.Hours()), int(span.Minutes())%60)
	}
	return fmt.Sprintf("%dd %dh", int(span.Hours())/24, int(span.Hours())%24)
}
