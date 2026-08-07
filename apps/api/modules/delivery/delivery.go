// Package delivery routes alerts to the configured targets and sends them.
package delivery

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/FacileStudio/Antenne/apps/api/internal/envelope"
	"github.com/FacileStudio/Antenne/apps/api/modules/settings"
)

// Message is one alert, in both the rendered and the structured form. HTML goes
// to the chat channels, the envelope to the machine ones.
type Message struct {
	HTML     string
	Envelope envelope.Envelope
}

// Attempt records what happened for one target.
type Attempt struct {
	TargetID   string `json:"targetId"`
	TargetName string `json:"targetName"`
	TargetType string `json:"targetType"`
	Status     string `json:"status"`
	Error      string `json:"error,omitempty"`
	DurationMS int64  `json:"durationMs"`
}

// Attempt statuses.
const (
	StatusSuccess = "success"
	StatusError   = "error"
)

// Route is what the routing rules selected for one event.
type Route struct {
	Selected            []settings.DeliveryTarget
	Sendable            []settings.DeliveryTarget
	SkippedByFilter     int
	SkippedUnconfigured []settings.DeliveryTarget
}

// ErrAllFailed is returned when every sendable target errored.
var ErrAllFailed = errors.New("delivery: all configured delivery channels failed")

// FailureError carries the per-target attempts alongside ErrAllFailed, so the
// caller can still log and retry each one.
type FailureError struct {
	Attempts []Attempt
}

func (e *FailureError) Error() string { return ErrAllFailed.Error() }

// Unwrap lets errors.Is(err, ErrAllFailed) match.
func (e *FailureError) Unwrap() error { return ErrAllFailed }

// Sender performs the deliveries. It holds the shared HTTP client and the
// Matrix session cache.
type Sender struct {
	log      *slog.Logger
	client   *http.Client
	matrixMu sync.Mutex
	matrix   map[string]*matrixSession
}

// NewSender builds a sender.
func NewSender(log *slog.Logger) *Sender {
	return &Sender{
		log:    log,
		client: &http.Client{Timeout: 30 * time.Second},
		matrix: map[string]*matrixSession{},
	}
}

// CanSend reports whether a target is configured well enough to try.
func CanSend(target settings.DeliveryTarget) bool {
	if !target.Enabled {
		return false
	}
	switch target.Type {
	case settings.TargetMatrix:
		hasCredential := target.AccessToken != "" ||
			(target.RefreshToken != nil && *target.RefreshToken != "") ||
			hasPasswordAuth(target)
		return target.Homeserver != "" && len(target.RoomIDs) > 0 && hasCredential
	case settings.TargetDiscord:
		return target.WebhookURL != ""
	case settings.TargetSMTP:
		return target.Host != "" && target.FromAddress != "" && len(target.ToAddresses) > 0
	case settings.TargetNtfy:
		return target.ServerURL != "" && target.Topic != ""
	case settings.TargetPerception:
		return target.URL != "" && target.Workspace != "" && target.SecretValue != nil && *target.SecretValue != ""
	case settings.TargetWebhook:
		return target.URL != ""
	}
	return false
}

// HasAny reports whether at least one target could receive anything.
func HasAny(config settings.DeliveryConfig) bool {
	for _, target := range config.Items {
		if CanSend(target) {
			return true
		}
	}
	return false
}

// HasConfiguredMatrix reports whether a usable Matrix target exists, which the
// dashboard surfaces as its own status flag.
func HasConfiguredMatrix(config settings.DeliveryConfig) bool {
	for _, target := range config.Items {
		if target.Type == settings.TargetMatrix && CanSend(target) {
			return true
		}
	}
	return false
}

// Accepts reports whether a target has opted into this provider.
//
// Routing is opt-in with no fallthrough: a target receives an event only if it
// names the provider id or shares one of its tags. A freshly created target
// receives nothing until one of the two is set, and Antenne's own events need the
// reserved `__antenne_system__` id listed explicitly.
func Accepts(target settings.DeliveryTarget, providerID string, providerTags []string) bool {
	if providerID != "" {
		for _, candidate := range target.ProviderIDs {
			if candidate == providerID {
				return true
			}
		}
	}

	if len(providerTags) == 0 || len(target.ProviderTags) == 0 {
		return false
	}
	for _, tag := range providerTags {
		for _, candidate := range target.ProviderTags {
			if candidate == tag {
				return true
			}
		}
	}
	return false
}

// Summarize resolves the routing rules for one event.
func Summarize(config settings.DeliveryConfig, providerID string, providerTags []string) Route {
	route := Route{}
	for _, target := range config.Items {
		if !Accepts(target, providerID, providerTags) {
			continue
		}
		route.Selected = append(route.Selected, target)
		if CanSend(target) {
			route.Sendable = append(route.Sendable, target)
		} else {
			route.SkippedUnconfigured = append(route.SkippedUnconfigured, target)
		}
	}
	route.SkippedByFilter = len(config.Items) - len(route.Selected)
	return route
}

// Send delivers a message to every target the routing rules selected, in
// parallel, and reports one attempt per target.
func (s *Sender) Send(ctx context.Context, config settings.DeliveryConfig, message Message, providerID string, providerTags []string) ([]Attempt, error) {
	if !HasAny(config) {
		return nil, errors.New("delivery: no delivery channel is configured")
	}

	targets := Summarize(config, providerID, providerTags).Sendable
	if len(targets) == 0 {
		return nil, nil
	}

	attempts := make([]Attempt, len(targets))
	var wait sync.WaitGroup

	for index, target := range targets {
		wait.Add(1)
		go func(index int, target settings.DeliveryTarget) {
			defer wait.Done()
			started := time.Now()
			err := s.SendTo(ctx, target, message)
			attempt := Attempt{
				TargetID:   target.ID,
				TargetName: target.Name,
				TargetType: string(target.Type),
				Status:     StatusSuccess,
				DurationMS: time.Since(started).Milliseconds(),
			}
			if err != nil {
				attempt.Status = StatusError
				attempt.Error = err.Error()
				s.log.Error("delivery failed",
					slog.String("target", target.Name),
					slog.String("target_type", string(target.Type)),
					slog.Any("error", err))
			}
			attempts[index] = attempt
		}(index, target)
	}

	wait.Wait()

	failures := 0
	for _, attempt := range attempts {
		if attempt.Status == StatusError {
			failures++
		}
	}
	if failures == len(attempts) {
		return attempts, &FailureError{Attempts: attempts}
	}

	return attempts, nil
}

// SendTo delivers a message to one target, whatever its type.
func (s *Sender) SendTo(ctx context.Context, target settings.DeliveryTarget, message Message) error {
	if !target.Type.IsValid() {
		return fmt.Errorf("delivery: unknown delivery target type %q", target.Type)
	}
	if !CanSend(target) {
		return fmt.Errorf("delivery: target `%s` is not fully configured", target.Name)
	}

	switch target.Type {
	case settings.TargetMatrix:
		return s.sendMatrix(ctx, target, message.HTML)
	case settings.TargetDiscord:
		return s.sendDiscord(ctx, target, message.HTML)
	case settings.TargetSMTP:
		return s.sendSMTP(target, message.HTML)
	case settings.TargetNtfy:
		return s.sendNtfy(ctx, target, message.HTML)
	case settings.TargetPerception:
		return s.sendPerception(ctx, target, message)
	}
	return s.sendHTTP(ctx, target, message)
}

func hasPasswordAuth(target settings.DeliveryTarget) bool {
	return target.UserID != "" && target.Password != nil && *target.Password != ""
}
