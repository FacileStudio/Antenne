/**
 * The eight health states, mirroring the Go contract exactly.
 *
 * `unknown` is not a synonym for `ok`: a generic webhook has no safe read-only
 * probe, so it stays unknown until a real delivery says otherwise, and painting
 * it green is the failure this feature exists to prevent. `probe_error` is
 * Antenne's own check failing, not the target refusing. `recovering` is
 * answering again with the outage not yet closed — never `down`, because a red
 * dot on a target the operator just watched work teaches them to ignore the dot.
 */
export type HealthState =
	| 'unknown'
	| 'ok'
	| 'pending'
	| 'down'
	| 'recovering'
	| 'flapping'
	| 'stale'
	| 'probe_error';

/** Which tier produced the state. They are not equally trustworthy. */
export type HealthSource = '' | 'delivery' | 'probe' | 'synthetic';

export type HealthKind =
	| ''
	| 'dead'
	| 'refreshable'
	| 'denied'
	| 'unreachable'
	| 'rate_limited'
	| 'unsupported';

export interface TargetHealth {
	targetId: string;
	name: string;
	type: string;
	enabled: boolean;
	state: HealthState;
	/** The circuit breaker's own answer, so the client never re-derives it. */
	deliverable: boolean;
	source: HealthSource;
	kind: HealthKind;
	/** The far side's own words. Rendered verbatim, never paraphrased. */
	detail: string;
	lastObservedAt: string | null;
	lastSuccessAt: string | null;
	lastFailureAt: string | null;
	downSince: string | null;
	/** Measured on the server's clock, so a skewed browser cannot invent an outage. */
	downtimeSeconds: number;
	probeSuspendedUntil: string | null;
	alertCount: number;
}

/** An alert no target could carry. Its existence is the outage's own evidence. */
export interface StandingAlert {
	targetId: string;
	targetName: string;
	title: string;
	detail: string;
	since: string;
}

export interface DeliveryHealthPage {
	targets: TargetHealth[];
	/** False when no enabled target can receive anything — a total outage. */
	anyDeliverable: boolean;
	standingAlerts: StandingAlert[] | null;
}

export interface TargetCheckResult {
	targetId: string;
	ok: boolean;
	kind: HealthKind;
	source: HealthSource;
	detail: string;
	/** The far side's own ask. Antenne's own limit arrives as an HTTP 429. */
	retryAfterSeconds: number;
	/** Where the target stands after the policy saw this observation. */
	state: HealthState;
}
