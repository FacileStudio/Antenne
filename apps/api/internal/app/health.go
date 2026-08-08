package app

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/FacileStudio/Antenne/apps/api/internal/envelope"
	"github.com/FacileStudio/Antenne/apps/api/modules/delivery"
	"github.com/FacileStudio/Antenne/apps/api/modules/events"
	"github.com/FacileStudio/Antenne/apps/api/modules/health"
	"github.com/FacileStudio/Antenne/apps/api/modules/settings"
)

// HealthDeps is everything the health subsystem needs from outside the pipeline.
//
// Records, Probe and Policy are passed in rather than constructed here so that
// the persistence, the per-provider probe calls and the decision policy each stay
// owned by the package that implements them, and so this file names no
// constructor it does not own.
type HealthDeps struct {
	Records health.Records
	Probe   health.ProbeFunc

	// Policy is health.Evaluate. Injecting it makes the wiring site the one place
	// that shows there is exactly one policy in the process.
	Policy health.Policy

	// Ping proves the database still answers, which is half of what the heartbeat
	// means by "functional".
	Ping func(ctx context.Context) error

	HeartbeatURL      string
	HeartbeatInterval time.Duration
}

// healthRuntime holds the three pieces and is nil-safe throughout, so an
// un-attached App behaves exactly as it did before this existed.
type healthRuntime struct {
	monitor   *health.Monitor
	escalator *health.Escalator
	heartbeat *health.Heartbeat
}

// AttachHealth builds the target-health subsystem. It must be called before
// Start; without it Antenne runs with no health monitoring at all.
func (a *App) AttachHealth(deps HealthDeps) {
	if deps.Records == nil || deps.Probe == nil || deps.Policy == nil {
		a.log.Warn("target health monitoring is not configured; delivery targets will not be probed")
		return
	}

	runtime := &healthRuntime{}

	runtime.escalator = health.NewEscalator(a.log,
		func() []settings.DeliveryTarget { return a.store.Get().Delivery.Items },
		func(ctx context.Context) map[string]health.State { return runtime.monitor.States(ctx) },
		a.sendHealthAlert,
		a.recordHealthNotice,
	)
	runtime.monitor = health.NewMonitor(a.log, deps.Records, deps.Probe, deps.Policy, runtime.escalator)
	runtime.heartbeat = health.NewHeartbeat(a.log, deps.HeartbeatURL, deps.HeartbeatInterval,
		a.vitals(runtime, deps.Ping, deps.HeartbeatInterval))

	a.health = runtime
}

// sendHealthAlert delivers a health notice straight to one target.
//
// Straight, and not through Dispatch: an alert about a broken channel must not be
// routed by the same rules that would have sent it to the broken channel, and a
// notice that never enters the pipeline can never enqueue a retry, exhaust it and
// produce a notice about itself.
func (a *App) sendHealthAlert(ctx context.Context, target settings.DeliveryTarget, messageHTML string) error {
	if !delivery.CanSend(target) {
		return fmt.Errorf("app: target `%s` is not fully configured", target.Name)
	}
	return a.sender.SendTo(ctx, target, delivery.Message{
		HTML: messageHTML,
		Envelope: envelope.Build(envelope.Input{
			Source:       envelope.SourceSystem,
			ProviderID:   envelope.Str(settings.SystemProviderID),
			ProviderType: "system",
			ProviderName: "Antenne",
			EventType:    "system.target.health",
			EventTitle:   "delivery target health",
			MessageHTML:  messageHTML,
			Subject:      &envelope.Subject{Type: envelope.SubjectSystem, Name: envelope.Str("Antenne")},
		}),
	})
}

// recordHealthNotice keeps every health alert in the activity log, including the
// ones nothing could carry — that row is the only local trace of a total outage.
func (a *App) recordHealthNotice(notice health.Notice) {
	route := "escalated via " + strings.Join(notice.DeliveredTo, ", ")
	if len(notice.DeliveredTo) == 0 {
		route = "no healthy target could carry this alert"
	}
	a.events.Push(events.New{
		Source:       string(envelope.SourceSystem),
		ProviderName: "Antenne",
		ProviderType: "health",
		Event:        fmt.Sprintf("target %s: %s", notice.Kind, notice.TargetName),
		Message:      notice.MessageHTML + "<br/><i>" + envelope.EscapeHTML(route) + "</i>",
	})
}

func (a *App) vitals(runtime *healthRuntime, ping func(context.Context) error, interval time.Duration) health.LivenessFunc {
	// The sweep runs once a minute, so anything shorter than a couple of minutes
	// would report a stall that is only a scheduling gap.
	staleAfter := interval
	if staleAfter < 2*time.Minute {
		staleAfter = 2 * time.Minute
	}

	return func(ctx context.Context) health.Vitals {
		if ping != nil {
			if err := ping(ctx); err != nil {
				return health.Vitals{Reason: "database is unreachable: " + err.Error()}
			}
		}

		last := runtime.monitor.LastSweepAt()
		if last.IsZero() {
			return health.Vitals{Reason: "the monitoring sweep has not completed a pass yet"}
		}
		if time.Since(last) > staleAfter {
			return health.Vitals{Reason: fmt.Sprintf("the monitoring sweep last completed %s ago", time.Since(last).Truncate(time.Second))}
		}

		degraded := false
		for _, state := range runtime.monitor.States(ctx) {
			if state == health.StateDown {
				degraded = true
				break
			}
		}
		return health.Vitals{Functional: true, Degraded: degraded}
	}
}

// TargetHealth is the persisted record for every target that has one. Targets
// with no record are absent rather than zero-valued; the HTTP layer fills them in
// as StateUnknown, which is the same fact.
func (a *App) TargetHealth() []health.Record {
	if a.health == nil {
		return nil
	}
	return a.health.monitor.Snapshot(context.Background())
}

// CheckTargetHealth probes one target on demand and records the outcome. A
// refusal by the far side is a returned Observation, not an error: the check ran.
// The error is reserved for the check being unable to run at all.
func (a *App) CheckTargetHealth(ctx context.Context, targetID string) (health.Observation, error) {
	if a.health == nil {
		return health.Observation{}, errors.New("app: target health monitoring is not configured")
	}
	target, ok := a.store.Get().FindTarget(targetID)
	if !ok {
		return health.Observation{}, errors.New("app: delivery target not found")
	}
	return a.health.monitor.Check(ctx, target), nil
}

// StandingHealthAlerts is every alert nothing could carry. The dashboard shows
// these as a banner: when every channel is down the dashboard is the only channel
// left, so it has to say so unprompted.
func (a *App) StandingHealthAlerts() []health.StandingAlert {
	if a.health == nil {
		return nil
	}
	return a.health.escalator.Standing()
}

// observeDeliveries feeds successful deliveries into health.
//
// Successes only, and on purpose. A real outcome is the most authoritative signal
// there is and it is free, so recording it is what lets the scheduler skip the
// probe for a busy target. Failures are not recorded here because the retry queue
// usually fixes them within seconds; the permanent verdict is
// notifyDeliveryExhausted's to give.
func (a *App) observeDeliveries(ctx context.Context, current settings.AppSettings, attempts []delivery.Attempt) {
	if a.health == nil {
		return
	}
	for _, attempt := range attempts {
		if attempt.Status != delivery.StatusSuccess {
			continue
		}
		target, ok := current.FindTarget(attempt.TargetID)
		if !ok {
			continue
		}
		a.health.monitor.Observe(ctx, target, health.Observation{
			OK:     true,
			Source: health.SourceDelivery,
		})
	}
}

func (r *healthRuntime) Apply(ctx context.Context, current settings.AppSettings) {
	if r == nil {
		return
	}
	r.monitor.Apply(ctx, current)
}

func (r *healthRuntime) Start(ctx context.Context) {
	if r == nil {
		return
	}
	r.heartbeat.Start(ctx)
}

func (r *healthRuntime) Stop() {
	if r == nil {
		return
	}
	r.monitor.Stop()
	r.heartbeat.Stop()
}

func (r *healthRuntime) observeFailure(target settings.DeliveryTarget, detail string, countAsLost bool) {
	if r == nil {
		return
	}
	if countAsLost {
		r.escalator.RecordUndelivered(target.ID)
	}
	r.monitor.Observe(context.Background(), target, health.Observation{
		OK:     false,
		Source: health.SourceDelivery,
		// KindUnreachable, not KindDead: the retry queue drops the underlying error
		// on the floor, so all this path knows is that three attempts failed. The
		// conservative kind never suspends probing on a guess of ours, and the probe
		// classifier will supply the real one on the next sweep.
		Kind:   health.KindUnreachable,
		Detail: detail,
	})
}
