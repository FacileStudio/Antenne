package server

import (
	"context"
	"math"
	"net/http"
	"strconv"
	"time"

	apierrors "github.com/FacileStudio/tronc/errors"
	"github.com/FacileStudio/tronc/httpjson"
	"github.com/go-chi/chi/v5"

	"github.com/FacileStudio/Antenne/apps/api/internal/app"
	"github.com/FacileStudio/Antenne/apps/api/internal/authcontext"
	"github.com/FacileStudio/Antenne/apps/api/modules/health"
	"github.com/FacileStudio/Antenne/apps/api/modules/settings"
)

// healthReporter is the slice of the pipeline these two endpoints need, named
// here rather than taken as *app.App wholesale so the dependency is one
// sentence long and the compile breaks at this line — not inside a handler — if
// the shape moves.
type healthReporter interface {
	// TargetHealth returns the persisted record for every target that has one.
	// Targets with no record at all are absent, not zero-valued: an absent row
	// and a row saying "unknown" are the same fact and the join below fills it.
	TargetHealth() []health.Record

	// CheckTargetHealth probes one target on demand and records the outcome.
	// A handshake the far side refused is a *returned Observation*, never an
	// error: the check ran, and reporting it as a failure of the request is how
	// a rate-limited diagnostic ends up displayed as "the target refused us".
	// The error is reserved for the check being unable to run at all.
	CheckTargetHealth(ctx context.Context, targetID string) (health.Observation, error)

	// StandingHealthAlerts is every alert no target could carry. When this is not
	// empty the alerting system has failed to alert, and the dashboard is the
	// only channel left.
	StandingHealthAlerts() []health.StandingAlert
}

var _ healthReporter = (*app.App)(nil)

// checkBudget is the on-demand check's own allowance, deliberately separate
// from every other endpoint's. Courrier shipped its check button on top of the
// folder-sync route, so pressing it spent a 5-per-minute budget shared with
// normal use, and the 429 that followed was reported to the operator as "IMAP
// refused the connection" — the diagnostic blaming the thing it was diagnosing.
// A diagnostic never shares a budget with real work.
//
// Ten a minute is one press per target on a large install, or a dozen impatient
// presses on one target, and it is well under the point where a provider starts
// counting: the danger is a held-down button, not a deliberate operator.
const (
	checkBudget = 10
	checkWindow = time.Minute
	checkBlock  = time.Minute
)

// TargetHealth is one dashboard row: the health record joined to the target it
// describes. A record is keyed by id and carries nothing else, so a client
// holding records alone would have to guess at names.
type TargetHealth struct {
	TargetID string `json:"targetId"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	Enabled  bool   `json:"enabled"`

	State string `json:"state"`

	// Deliverable is the circuit breaker's answer, sent rather than inferred:
	// the mapping from state to "will the router try" lives in one place and
	// this is it.
	Deliverable bool `json:"deliverable"`

	// Source says which tier produced the state, because a real delivery and a
	// probe are not equally authoritative and the dashboard has to say which.
	Source string `json:"source"`
	Kind   string `json:"kind"`

	// Detail is the far side's own words, verbatim. It is what an operator
	// pastes into a search or forwards to their provider, so nothing here or
	// downstream paraphrases it.
	Detail string `json:"detail"`

	LastObservedAt *time.Time `json:"lastObservedAt"`
	LastSuccessAt  *time.Time `json:"lastSuccessAt"`
	LastFailureAt  *time.Time `json:"lastFailureAt"`
	DownSince      *time.Time `json:"downSince"`

	// DowntimeSeconds is computed against the server's clock. "It broke at some
	// point" is not actionable, and a browser whose clock is off by an hour
	// would invent a different outage from the same timestamp.
	DowntimeSeconds int `json:"downtimeSeconds"`

	ProbeSuspendedUntil *time.Time `json:"probeSuspendedUntil"`
	AlertCount          int        `json:"alertCount"`
}

// DeliveryHealthPage is the full health snapshot for every configured delivery
// target, plus the boolean that says whether any target can currently receive
// anything and the standing alerts that the alerting system could not carry.
type DeliveryHealthPage struct {
	Targets []TargetHealth `json:"targets"`

	// AnyDeliverable is false when no enabled target can currently receive
	// anything. That is the case where Antenne cannot tell anyone anything, so
	// the dashboard is the only channel left and it has to say so unprompted.
	AnyDeliverable bool `json:"anyDeliverable"`

	// StandingAlerts are the notices that outage actually swallowed. The boolean
	// above says the channels are gone; these say what nobody was told.
	StandingAlerts []health.StandingAlert `json:"standingAlerts"`
}

// TargetCheckResult is the response to one on-demand probe: the verbatim result
// from the far side, the derived state after policy, and the retry budget
// consumed (only when the far side asked for one).
type TargetCheckResult struct {
	TargetID string `json:"targetId"`
	OK       bool   `json:"ok"`
	Kind     string `json:"kind"`
	Source   string `json:"source"`
	Detail   string `json:"detail"`

	// RetryAfterSeconds is the *far side's* binding ask, when it sent one. Ours
	// travels as the HTTP Retry-After header on a 429, so the two can never be
	// mistaken for each other.
	RetryAfterSeconds int `json:"retryAfterSeconds"`

	// State is where the target stands after this observation went through the
	// policy, which is not the same as this observation: one failure inside the
	// hysteresis threshold leaves a target `pending`, not `down`.
	State string `json:"state"`
}

func (a *API) handleTargetHealth(w http.ResponseWriter, r *http.Request) {
	current := a.app.Settings()
	page := buildHealthPage(current.Delivery.Items, a.app.TargetHealth(), time.Now())
	page.StandingAlerts = a.app.StandingHealthAlerts()
	httpjson.WriteJSON(w, http.StatusOK, page)
}

// handleCheckTargetHealth is the operator's "check this now" button.
func (a *API) handleCheckTargetHealth(w http.ResponseWriter, r *http.Request) {
	client := authcontext.ClientIP(r)
	if wait := a.checks.RetryAfter(client); wait > 0 {
		// 429 with Retry-After, so the client can say "retry in Ns" about
		// Antenne instead of attributing a limit we imposed to the target.
		w.Header().Set("Retry-After", strconv.Itoa(int(math.Ceil(wait.Seconds()))))
		httpjson.WriteError(w, apierrors.RateLimited("too many checks from this client; wait before checking again"))
		return
	}
	// Every call spends budget, not only the ones that fail: the cost being
	// limited here is the outbound handshake, which happens either way.
	a.checks.Fail(client)

	targetID := chi.URLParam(r, "targetId")
	current := a.app.Settings()
	target, ok := current.FindTarget(targetID)
	if !ok {
		httpjson.WriteError(w, apierrors.NotFound("delivery target not found"))
		return
	}

	observation, err := a.app.CheckTargetHealth(r.Context(), targetID)
	if err != nil {
		// The check could not run. Not the target's verdict, so not the target's
		// fault, so not a 200 either.
		httpjson.WriteError(w, apierrors.Internal("the check could not run: "+err.Error(), err))
		return
	}

	httpjson.WriteJSON(w, http.StatusOK, TargetCheckResult{
		TargetID:          targetID,
		OK:                observation.OK,
		Kind:              string(observation.Kind),
		Source:            string(observation.Source),
		Detail:            observation.Detail,
		RetryAfterSeconds: int(observation.RetryAfter.Seconds()),
		State:             describeHealth(target, recordFor(a.app.TargetHealth(), targetID), time.Now()).State,
	})
}

func buildHealthPage(targets []settings.DeliveryTarget, records []health.Record, now time.Time) DeliveryHealthPage {
	page := DeliveryHealthPage{Targets: make([]TargetHealth, 0, len(targets))}
	for _, target := range targets {
		row := describeHealth(target, recordFor(records, target.ID), now)
		if row.Enabled && row.Deliverable {
			page.AnyDeliverable = true
		}
		page.Targets = append(page.Targets, row)
	}
	return page
}

func recordFor(records []health.Record, targetID string) health.Record {
	for _, record := range records {
		if record.TargetID == targetID {
			return record
		}
	}
	// Never measured. StateUnknown is the honest answer and the zero State is
	// not it, so say so explicitly rather than shipping an empty string a client
	// has to guess at.
	return health.Record{TargetID: targetID, State: health.StateUnknown}
}

func describeHealth(target settings.DeliveryTarget, record health.Record, now time.Time) TargetHealth {
	state := record.State
	if state == "" {
		state = health.StateUnknown
	}
	return TargetHealth{
		TargetID:            target.ID,
		Name:                target.Name,
		Type:                string(target.Type),
		Enabled:             target.Enabled,
		State:               string(state),
		Deliverable:         state.Deliverable(),
		Source:              string(record.LastSource),
		Kind:                string(record.LastKind),
		Detail:              record.LastDetail,
		LastObservedAt:      record.LastObservedAt,
		LastSuccessAt:       record.LastSuccessAt,
		LastFailureAt:       record.LastFailureAt,
		DownSince:           record.DownSince,
		DowntimeSeconds:     int(record.Downtime(now).Seconds()),
		ProbeSuspendedUntil: record.ProbeSuspendedUntil,
		AlertCount:          record.AlertCount,
	}
}
