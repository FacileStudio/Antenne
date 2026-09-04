package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/FacileStudio/Antenne/apps/api/internal/envelope"
	"github.com/FacileStudio/Antenne/apps/api/modules/delivery"
	"github.com/FacileStudio/Antenne/apps/api/modules/events"
	"github.com/FacileStudio/Antenne/apps/api/modules/providers"
	"github.com/FacileStudio/Antenne/apps/api/modules/settings"
	"github.com/FacileStudio/Antenne/apps/api/modules/webhooks"
)

func (a *App) fireProviderEvents(ctx context.Context, previous, next settings.AppSettings) {
	switches := next.System.Events

	previousByID := map[string]settings.Provider{}
	for _, provider := range previous.Providers.Items {
		previousByID[provider.ID] = provider
	}
	nextByID := map[string]settings.Provider{}
	for _, provider := range next.Providers.Items {
		nextByID[provider.ID] = provider
	}

	for _, provider := range next.Providers.Items {
		before, existed := previousByID[provider.ID]
		switch {
		case !existed && switches.ProviderAdded:
			a.fireSystemEvent(ctx,
				fmt.Sprintf("<b>Provider added</b><br/><b>%s</b> (%s) was added.",
					envelope.EscapeHTML(provider.Name), envelope.EscapeHTML(string(provider.Type))),
				"provider added: "+provider.Name, "system.provider.added")
		case existed && provider.Enabled && !before.Enabled && switches.ProviderEnabled:
			a.fireSystemEvent(ctx,
				fmt.Sprintf("<b>Provider enabled</b><br/><b>%s</b> is now active.", envelope.EscapeHTML(provider.Name)),
				"provider enabled: "+provider.Name, "system.provider.enabled")
		case existed && !provider.Enabled && before.Enabled && switches.ProviderDisabled:
			a.fireSystemEvent(ctx,
				fmt.Sprintf("<b>Provider disabled</b><br/><b>%s</b> is now inactive.", envelope.EscapeHTML(provider.Name)),
				"provider disabled: "+provider.Name, "system.provider.disabled")
		}
	}

	if !switches.ProviderRemoved {
		return
	}
	for _, provider := range previous.Providers.Items {
		if _, ok := nextByID[provider.ID]; ok {
			continue
		}
		a.fireSystemEvent(ctx,
			fmt.Sprintf("<b>Provider removed</b><br/><b>%s</b> (%s) was removed.",
				envelope.EscapeHTML(provider.Name), envelope.EscapeHTML(string(provider.Type))),
			"provider removed: "+provider.Name, "system.provider.removed")
	}
}

func (a *App) fireSystemEvent(ctx context.Context, message, event, eventType string) {
	_, err := a.Dispatch(ctx, Alert{
		MessageHTML:  message,
		Source:       envelope.SourceSystem,
		ProviderID:   settings.SystemProviderID,
		ProviderName: "Antenne",
		ProviderType: "system",
		Event:        event,
		EventType:    eventType,
		Subject:      &envelope.Subject{Type: envelope.SubjectSystem, Name: envelope.Str("Antenne")},
	})
	if err != nil {
		a.log.Error("system event was not delivered",
			slog.String("event", event), slog.Any("error", err))
	}
}

// Dispatch is the single path an alert takes: dedup, envelope, log, route,
// deliver, retry.
func (a *App) Dispatch(ctx context.Context, alert Alert) ([]delivery.Attempt, error) {
	// Detached from the caller's cancellation. Most callers hand in a request
	// context: the webhook intake's, or a settings save's. A third party whose
	// HTTP client gives up early would otherwise cancel the Matrix send it just
	// triggered, and the event would be logged as failed for no reason. Each
	// sender still has its own 30-second timeout, so nothing hangs.
	ctx = context.WithoutCancel(ctx)

	current := a.store.Get()

	if !delivery.HasAny(current.Delivery) {
		return nil, errors.New("no delivery target configured; add and enable at least one delivery target first")
	}

	providerID := alert.ProviderID
	if providerID == "" {
		providerID = settings.SystemProviderID
	}
	tags := alert.ProviderTags
	if tags == nil {
		tags = current.ProviderTags(providerID)
	}

	label := alert.Event
	if label == "" {
		label = truncate(envelope.HTMLToPlainText(alert.MessageHTML), 80)
	}

	if a.isDuplicate(alert.MessageHTML, providerID) {
		a.log.Debug("event deduplicated", slog.String("event", label))
		return nil, nil
	}

	built := envelope.Build(envelope.Input{
		Source:            defaultSource(alert.Source),
		ProviderID:        envelope.Str(providerID),
		ProviderType:      defaultString(alert.ProviderType, "system"),
		ProviderName:      defaultString(alert.ProviderName, "Antenne"),
		ProviderTags:      tags,
		EventType:         defaultString(alert.EventType, label),
		EventTitle:        defaultString(alert.EventTitle, label),
		MessageHTML:       alert.MessageHTML,
		EventURL:          alert.EventURL,
		Actor:             alert.Actor,
		Subject:           alert.Subject,
		NormalizedPayload: alert.Normalized,
		RawPayload:        alert.RawPayload,
		OccurredAt:        alert.OccurredAt,
		ContainsPII:       alert.ContainsPII,
		RawPayloadAllowed: alert.RawPayloadAllowed,
	})

	entry := a.events.Push(events.New{
		Source:       string(built.Source),
		ProviderID:   envelope.Str(providerID),
		ProviderName: built.Provider.Name,
		ProviderType: built.Provider.Type,
		Event:        label,
		Message:      alert.MessageHTML,
		Envelope:     &built,
	})

	route := delivery.Summarize(current.Delivery, providerID, tags)
	sourceLabel := fmt.Sprintf("%s (%s:%s)", built.Provider.Name, built.Provider.Type, providerID)

	if len(route.Selected) == 0 {
		a.log.Info("event not delivered: no target selected this provider",
			slog.String("event", label), slog.String("source", sourceLabel))
		return nil, nil
	}
	if len(route.Sendable) == 0 {
		// Say which of the two it is. A target that is switched off is a
		// decision, and reporting it as a configuration fault sends the
		// reader to check a URL and a secret that were never wrong.
		if len(route.SkippedUnconfigured) == 0 {
			a.log.Info("event not delivered: every target for it is switched off",
				slog.String("event", label),
				slog.String("source", sourceLabel),
				slog.String("targets", describeTargets(route.SkippedDisabled)))
			return nil, nil
		}
		a.log.Warn("event not delivered: the selected targets are not fully configured",
			slog.String("event", label),
			slog.String("source", sourceLabel),
			slog.String("targets", describeTargets(route.SkippedUnconfigured)),
			slog.String("also_disabled", describeTargets(route.SkippedDisabled)))
		return nil, nil
	}

	message := delivery.Message{HTML: alert.MessageHTML, Envelope: built}
	attempts, err := a.sender.Send(ctx, current.Delivery, message, providerID, tags)

	var failure *delivery.FailureError
	if errors.As(err, &failure) {
		attempts = failure.Attempts
	} else if err != nil {
		return nil, err
	}

	a.events.SetDeliveries(entry.ID, attempts)
	a.observeDeliveries(ctx, current, attempts)

	failed := 0
	for _, attempt := range attempts {
		if attempt.Status != delivery.StatusError {
			continue
		}
		failed++
		if target, ok := current.FindTarget(attempt.TargetID); ok {
			a.retries.Enqueue(entry.ID, target, message)
		}
	}

	a.log.Info("event delivered",
		slog.String("event", label),
		slog.String("source", sourceLabel),
		slog.Int("delivered", len(attempts)-failed),
		slog.Int("failed", failed),
		slog.Int("filtered", route.SkippedByFilter))

	if failed > 0 && current.System.Events.DeliveryFailure {
		a.notifyDeliveryFailure(ctx, label, attempts)
	}

	if failure != nil {
		return attempts, failure
	}
	return attempts, nil
}

// DispatchBusEvent parses an event payload pushed on the bus through the
// corresponding provider definition and delivers it to active delivery targets.
func (a *App) DispatchBusEvent(ctx context.Context, providerType settings.ProviderType, body []byte) {
	current := a.store.Get()
	var (
		providerID   string
		providerName string
		providerTags []string
	)
	for _, p := range current.Providers.Items {
		if p.Enabled && p.Type == providerType {
			providerID = p.ID
			providerName = p.Name
			providerTags = p.Tags
			break
		}
	}
	if providerName == "" {
		providerName = settings.ProviderLabel(providerType)
	}

	definition, known := webhooks.Lookup(providerType)
	if !known {
		return
	}

	result, err := definition.Handle(body, webhooks.Context{
		ProviderID:   providerID,
		ProviderName: providerName,
		ProviderType: providerType,
	})
	if err != nil || result.MessageHTML == "" {
		return
	}

	var raw any
	if result.RawPayloadAllowed {
		raw = body
	}

	if _, err := a.Dispatch(ctx, Alert{
		Source:            envelope.Source("pool"),
		ProviderID:        providerID,
		ProviderType:      string(providerType),
		ProviderName:      providerName,
		ProviderTags:      providerTags,
		Event:             result.Description,
		EventType:         result.EventType,
		EventTitle:        result.EventTitle,
		EventURL:          result.EventURL,
		MessageHTML:       result.MessageHTML,
		OccurredAt:        result.OccurredAt,
		Actor:             result.Actor,
		Subject:           result.Subject,
		Normalized:        result.Normalized,
		RawPayload:        raw,
		ContainsPII:       result.ContainsPII,
		RawPayloadAllowed: result.RawPayloadAllowed,
	}); err != nil {
		a.log.Warn("bus event dispatch failed", slog.Any("error", err), slog.String("provider", providerName))
	}
}

func (a *App) notifyDeliveryFailure(ctx context.Context, label string, attempts []delivery.Attempt) {
	a.notifyingFailure.Lock()
	if a.inFailureNotice {
		a.notifyingFailure.Unlock()
		return
	}
	a.inFailureNotice = true
	a.notifyingFailure.Unlock()

	defer func() {
		a.notifyingFailure.Lock()
		a.inFailureNotice = false
		a.notifyingFailure.Unlock()
	}()

	names := make([]string, 0)
	for _, attempt := range attempts {
		if attempt.Status == delivery.StatusError {
			names = append(names, attempt.TargetName)
		}
	}

	// This fires on the first attempt, before the retry queue has run any of its
	// three, and in practice almost every one of these recovers within seconds.
	// So it says what actually happened rather than declaring a failure the
	// event log then quietly rewrites to success — the permanent verdict is
	// notifyDeliveryExhausted's to give.
	a.fireSystemEvent(ctx,
		fmt.Sprintf("<b>Delivery failed, retrying</b><br/>%d target(s) failed for <b>%s</b>: %s",
			len(names), envelope.EscapeHTML(label), envelope.EscapeHTML(strings.Join(names, ", "))),
		"delivery retrying", "system.delivery.retrying")
}

// notifyDeliveryExhausted records a delivery the retry queue has given up on.
//
// It no longer alerts anyone, and that is the behaviour change that matters most
// here. Alerting per undelivered event meant one dead webhook and fifty events
// produced fifty identical notices, while a dead webhook with no traffic produced
// none at all — the unit of a health notification is the target. So this keeps the
// event-log row and hands the outcome to health, which counts it towards the
// target's blast radius and lets the policy decide whether a human hears about it.
func (a *App) notifyDeliveryExhausted(target settings.DeliveryTarget, message delivery.Message, cause error) {
	label := "an event"
	if message.Envelope.Event.Title != nil && *message.Envelope.Event.Title != "" {
		label = *message.Envelope.Event.Title
	}

	// A health alert is sent straight to its recipients and never enters the
	// pipeline, so it cannot reach this path. Antenne's older per-event delivery
	// notices still route like any other event, and one of those failing must not
	// be counted as lost payload on top of the event it was already reporting.
	systemNotice := strings.HasPrefix(message.Envelope.Event.Type, "system.delivery.")
	if systemNotice {
		a.log.Error("a delivery notice could not be delivered either",
			slog.String("target", target.Name))
	}

	a.events.Push(events.New{
		Source:       string(envelope.SourceSystem),
		ProviderName: "Antenne",
		ProviderType: "health",
		Event:        "delivery failure: " + target.Name,
		Message: fmt.Sprintf("<b>Delivery failure</b><br/><b>%s</b> could not be delivered to <b>%s</b> after %d attempts",
			envelope.EscapeHTML(label), envelope.EscapeHTML(target.Name), delivery.MaxRetryAttempts),
	})

	// The far side's own words, not ours. "gave up after 3 attempts" describes
	// what Antenne did; the operator needs what the server said, because that is
	// the string they search for or forward to their provider.
	//
	// Through SafeErrorText, because a transport failure's error carries the URL
	// it was dialling and a Discord webhook URL contains its token — and this
	// string ends up in an alert sent to real rooms.
	detail := fmt.Sprintf("gave up after %d delivery attempts", delivery.MaxRetryAttempts)
	if cause != nil {
		detail = fmt.Sprintf("%s: %s", detail, delivery.SafeErrorText(cause))
	}

	a.health.observeFailure(target, detail, !systemNotice)
}

// emitFromProvider adapts a poller's alert into the pipeline.
func (a *App) emitFromProvider(source envelope.Source) providers.Emit {
	return func(alert providers.Alert) {
		_, err := a.Dispatch(context.Background(), Alert{
			MessageHTML:  alert.MessageHTML,
			Source:       source,
			ProviderID:   alert.ProviderID,
			ProviderName: alert.ProviderName,
			ProviderType: alert.ProviderType,
			Event:        alert.Event,
			EventType:    alert.EventType,
			EventTitle:   alert.EventTitle,
			EventURL:     alert.EventURL,
			OccurredAt:   alert.OccurredAt,
			Actor:        alert.Actor,
			Subject:      alert.Subject,
			Normalized:   alert.Normalized,
			ContainsPII:  alert.ContainsPII,
		})
		if err != nil {
			a.log.Error("provider alert was not delivered",
				slog.String("provider", alert.ProviderName), slog.Any("error", err))
		}
	}
}

func (a *App) isDuplicate(message, providerID string) bool {
	sum := sha256.Sum256([]byte(providerID + "\x00" + message))
	key := hex.EncodeToString(sum[:])
	now := time.Now()

	a.dedupMu.Lock()
	defer a.dedupMu.Unlock()

	for candidate, seen := range a.dedup {
		if now.Sub(seen) > dedupWindow {
			delete(a.dedup, candidate)
		}
	}

	if _, seen := a.dedup[key]; seen {
		return true
	}
	a.dedup[key] = now
	return false
}

func describeTargets(targets []settings.DeliveryTarget) string {
	if len(targets) == 0 {
		return "none"
	}
	parts := make([]string, 0, len(targets))
	for _, target := range targets {
		parts = append(parts, fmt.Sprintf("%s(%s)", target.Name, target.Type))
	}
	return strings.Join(parts, ", ")
}

func defaultSource(source envelope.Source) envelope.Source {
	if source == "" {
		return envelope.SourceSystem
	}
	return source
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func truncate(value string, limit int) string {
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit])
}
