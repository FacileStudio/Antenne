package health

import (
	"context"
	"log/slog"
	"math/rand/v2"
	"net/http"
	"strings"
	"sync"
	"time"
)

// DefaultHeartbeatInterval is the period used when HEARTBEAT_INTERVAL is unset.
//
// The receiver's own resolution bounds anything useful here: Grafana OnCall runs
// its checker every 13 minutes and therefore recommends 15 minutes or more, while
// healthchecks.io has no such floor. Five minutes suits healthchecks.io and Dead
// Man's Snitch; an OnCall user has to raise it, which is why this is configurable
// rather than a constant.
const DefaultHeartbeatInterval = 5 * time.Minute

// Vitals is the answer to "is this process actually doing its job", asked
// immediately before every ping.
type Vitals struct {
	// Functional is the whole point. A heartbeat that a zombie process can still
	// send proves that its TCP stack works and nothing else, so this must mean
	// end-to-end liveness: the database answered, and the monitoring sweep has
	// completed a pass recently.
	Functional bool

	// Degraded is true when at least one delivery target is down. It turns the
	// ping into a /fail, so the external side alarms on silence *and* on
	// self-reported degradation rather than only on the first.
	Degraded bool

	// Reason explains a non-functional verdict, for the local log.
	Reason string
}

// LivenessFunc reports the vitals. It is injected because everything it has to
// ask about — the database handle, the sweep timestamp, the target states — lives
// outside this package.
type LivenessFunc func(ctx context.Context) Vitals

// Heartbeat is the dead man's switch: an outward ping to a third party that
// alarms when it stops arriving.
//
// It is the only part of this subsystem that can report the two failures nothing
// in-process can. An in-process probe cannot tell anyone the process is dead, and
// cross-channel escalation cannot tell anyone every channel is down. Both are
// silence, and silence is what an external receiver is for.
type Heartbeat struct {
	log      *slog.Logger
	url      string
	interval time.Duration
	client   *http.Client
	vitals   LivenessFunc
	random   func() float64

	mu     sync.Mutex
	cancel context.CancelFunc
	group  sync.WaitGroup
}

// NewHeartbeat builds the switch, or returns nil when no URL is configured.
// Absent configuration turns the feature off silently: it is opt-in, and an
// operator who has not set up a receiver does not need to be told on every boot.
func NewHeartbeat(log *slog.Logger, url string, interval time.Duration, vitals LivenessFunc) *Heartbeat {
	url = strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(url), "/"))
	if url == "" {
		return nil
	}
	if interval <= 0 {
		interval = DefaultHeartbeatInterval
	}
	return &Heartbeat{
		log:      log,
		url:      url,
		interval: interval,
		client:   &http.Client{Timeout: 10 * time.Second},
		vitals:   vitals,
		random:   rand.Float64,
	}
}

// Interval is the configured period, which is what the receiver's period and
// grace time have to be set from.
func (h *Heartbeat) Interval() time.Duration {
	if h == nil {
		return 0
	}
	return h.interval
}

// Start begins pinging until ctx is cancelled. Nil-safe, so the caller does not
// have to branch on whether the feature is configured.
func (h *Heartbeat) Start(ctx context.Context) {
	if h == nil {
		return
	}

	h.Stop()
	loopCtx, cancel := context.WithCancel(ctx)
	h.mu.Lock()
	h.cancel = cancel
	h.mu.Unlock()

	h.log.Info("heartbeat enabled", slog.Duration("interval", h.interval))

	h.group.Add(1)
	go h.loop(loopCtx)
}

// Stop halts the pings.
func (h *Heartbeat) Stop() {
	if h == nil {
		return
	}
	h.mu.Lock()
	cancel := h.cancel
	h.cancel = nil
	h.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	h.group.Wait()
}

func (h *Heartbeat) loop(ctx context.Context) {
	defer h.group.Done()

	for {
		h.beat(ctx)

		timer := time.NewTimer(NextBeatIn(h.interval, h.random))
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}

func (h *Heartbeat) beat(ctx context.Context) {
	vitals := Vitals{Functional: true}
	if h.vitals != nil {
		vitals = h.vitals(ctx)
	}

	if !vitals.Functional {
		// Deliberately no ping at all, not even /fail. The receiver's grace timer
		// is the signal, and a process too broken to judge itself is exactly the
		// case where its own self-report should not be trusted.
		h.log.Error("heartbeat withheld: the service is not functional",
			slog.String("reason", vitals.Reason))
		return
	}

	url := h.url
	if vitals.Degraded {
		url += "/fail"
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		h.log.Error("heartbeat request", slog.Any("error", err))
		return
	}
	request.Header.Set("User-Agent", "Antenne/1.0 heartbeat")

	response, err := h.client.Do(request)
	if err != nil {
		if ctx.Err() != nil {
			return
		}
		h.log.Warn("heartbeat could not be sent", slog.Any("error", err))
		return
	}
	response.Body.Close()

	if response.StatusCode >= 400 {
		h.log.Warn("heartbeat receiver rejected the ping",
			slog.Int("status", response.StatusCode), slog.Bool("degraded", vitals.Degraded))
		return
	}
	h.log.Debug("heartbeat sent", slog.Bool("degraded", vitals.Degraded))
}

// NextBeatIn is the wait before the next ping: uniform in [0.5, 0.9) of the
// interval.
//
// Downward only, and that asymmetry is the single most common heartbeat bug.
// Symmetric jitter sends some pings *late*, which spends the receiver's grace
// time on our own randomness and manufactures "late" alarms about a service that
// is perfectly healthy. Pinging early costs nothing but an extra request.
func NextBeatIn(interval time.Duration, random func() float64) time.Duration {
	if interval <= 0 {
		interval = DefaultHeartbeatInterval
	}
	if random == nil {
		return interval / 2
	}
	return time.Duration(float64(interval) * (0.5 + 0.4*random()))
}
