import { goto } from '$app/navigation';
import type {
	AppSettings,
	DeliveryHealthPage,
	EventEntry,
	EventStats,
	BusStats,
	TargetCheckResult,
	SessionPayload,
	AuthConfigPayload,
	SettingsPayload,
	Space,
	SpaceMember
} from './types';

/**
 * ApiError carries the API's own error code alongside the message, so a caller
 * can branch on `conflict` or `unauthenticated` without parsing prose.
 */
export class ApiError extends Error {
	readonly status: number;
	readonly code: string;

	/**
	 * Seconds the server asked us to wait, from its Retry-After header; zero when
	 * it sent none. Without this a 429 reaches the UI as an indistinguishable
	 * failure, and a diagnostic button then reports Antenne's own rate limit as
	 * the remote end refusing the connection.
	 */
	readonly retryAfterSeconds: number;

	constructor(message: string, status: number, code: string, retryAfterSeconds = 0) {
		super(message);
		this.name = 'ApiError';
		this.status = status;
		this.code = code;
		this.retryAfterSeconds = retryAfterSeconds;
	}
}

/**
 * Reads the suite error envelope, {"error":{"code","message"}}.
 *
 * The body is read as text and parsed defensively rather than through
 * response.json(): the SPA catch-all answers an unknown path with 200 and HTML,
 * so a mistyped URL would otherwise throw a bare SyntaxError out of here with
 * nothing pointing at the cause.
 */
function parseError(body: string, response: Response): ApiError {
	const status = response.status;
	const wait = retryAfterSeconds(response);
	try {
		const payload = JSON.parse(body) as {
			error?: { code?: string; message?: string } | string;
			message?: string;
		};
		if (typeof payload.error === 'string') {
			return new ApiError(payload.error, status, 'unknown', wait);
		}
		const message = payload.error?.message ?? payload.message;
		if (message) {
			return new ApiError(message, status, payload.error?.code ?? 'unknown', wait);
		}
	} catch {
		// fall through to the status-only message
	}
	return new ApiError(`HTTP ${status}`, status, 'unknown', wait);
}

/** Retry-After in delta-seconds. The HTTP-date form is not used by this API. */
function retryAfterSeconds(response: Response): number {
	const raw = response.headers.get('Retry-After');
	if (!raw) return 0;
	const seconds = Number(raw);
	return Number.isFinite(seconds) && seconds > 0 ? Math.ceil(seconds) : 0;
}

/**
 * Methods a browser can be tricked into issuing cross-site without script.
 * Everything else has to carry the CSRF header.
 */
const SAFE_METHODS = new Set(['GET', 'HEAD', 'OPTIONS']);

/**
 * porte reads the session cookie before the Authorization header, and refuses a
 * mutating request authenticated by cookie unless this header is present. Any
 * value counts — the header's presence is the whole signal, because a browser
 * will not attach it to a simple cross-site request. There is no token to
 * distribute and none to rotate.
 *
 * Sending it on every write rather than only where it is required is what keeps
 * the failure mode from being invisible: without it every read succeeds and
 * every write 403s, which reads as "the app works, saving is broken" and cost
 * two other suite apps several hours in production.
 */
async function request<T>(path: string, init?: RequestInit): Promise<T> {
	const method = (init?.method ?? 'GET').toUpperCase();
	const headers: Record<string, string> = {
		'Content-Type': 'application/json',
		...((init?.headers as Record<string, string>) ?? {})
	};
	if (!SAFE_METHODS.has(method)) {
		headers['X-Facile-CSRF'] = '1';
	}

	const response = await fetch(`/api${path}`, {
		...init,
		credentials: 'include',
		headers
	});

	const body = await response.text();

	if (!response.ok) {
		throw parseError(body, response);
	}

	return body ? (JSON.parse(body) as T) : (undefined as T);
}

/**
 * A 401 is not handled here on purpose: POST /login answers 401 for a wrong
 * password, and a global redirect would hijack the login form's own error. The
 * app layout catches it instead, where the distinction is knowable.
 */
export function isUnauthenticated(error: unknown): boolean {
	return error instanceof ApiError && error.status === 401;
}

/**
 * A 429 is Antenne throttling this browser, and nothing at all about whatever
 * the request was pointed at. Callers must be able to tell the two apart before
 * they write a message naming a third party.
 */
export function isRateLimited(error: unknown): boolean {
	return error instanceof ApiError && error.status === 429;
}

export async function redirectToLogin() {
	await goto('/login', { replaceState: true });
}

export const api = {
	session: () => request<SessionPayload>('/session'),

	/**
	 * What this instance accepts, before holding any credential. porte owns the
	 * route and serves sso_only and oidc_enabled; password_required, username
	 * and email are Antenne's own additions to it.
	 */
	authConfig: () => request<AuthConfigPayload>('/auth/config'),

	logout: () => request<{ logged_out: boolean }>('/auth/logout', { method: 'POST', body: '{}' }),

	/**
	 * The signed-in person's pending Registre space invitations, for the user menu. porte/kit
	 * serves it only when single sign-on is configured, so a 404 means "no row", like a count of 0.
	 */
	invitations: () => request<{ count: number; href?: string }>('/me/invitations'),

	settings: () => request<SettingsPayload>('/settings'),
	saveSettings: (settings: AppSettings) =>
		request<SettingsPayload>('/settings', { method: 'PUT', body: JSON.stringify(settings) }),

	testAlert: () => request<{ ok: boolean }>('/test-alert', { method: 'POST', body: '{}' }),
	testTarget: (targetId: string) =>
		request<{ ok: boolean }>('/test-delivery-target', {
			method: 'POST',
			body: JSON.stringify({ targetId })
		}),
	testProvider: (providerId: string) =>
		request<{ ok: boolean }>('/test-provider-connection', {
			method: 'POST',
			body: JSON.stringify({ providerId })
		}),

	deliveryHealth: () => request<DeliveryHealthPage>('/delivery-health'),

	/**
	 * The operator-triggered probe. It answers 200 with `ok: false` when the far
	 * side declines — the check itself worked — so a caller must read the body,
	 * not the status, for the target's verdict. A non-2xx here is about the
	 * request: 429 for this endpoint's own budget, 401 for the session.
	 */
	checkDeliveryTarget: (targetId: string) =>
		request<TargetCheckResult>(`/delivery-health/${encodeURIComponent(targetId)}/check`, {
			method: 'POST',
			body: '{}'
		}),

	events: (options: {
		limit?: number;
		offset?: number;
		q?: string;
		source?: string;
		providerId?: string;
		targetId?: string;
	}) => {
		const params = new URLSearchParams();
		if (options.limit) params.set('limit', String(options.limit));
		if (options.offset) params.set('offset', String(options.offset));
		if (options.q) params.set('q', options.q);
		if (options.source) params.set('source', options.source);

		let path = '/events';
		if (options.providerId) path = `/events/provider/${encodeURIComponent(options.providerId)}`;
		else if (options.targetId) path = `/events/target/${encodeURIComponent(options.targetId)}`;

		const query = params.toString();
		return request<{ events: EventEntry[]; total: number; stats: EventStats }>(
			query ? `${path}?${query}` : path
		);
	},

	replay: (eventId: string, targetId: string) =>
		request<{ ok: boolean }>(`/events/${encodeURIComponent(eventId)}/replay`, {
			method: 'POST',
			body: JSON.stringify({ targetId })
		}),

	busStats: () => request<BusStats>('/antenne/stats'),

	spaces: () => request<{ spaces: Space[]; manageUrl: string }>('/spaces'),
	space: (id: string) =>
		request<{ space: Space; members: SpaceMember[] }>(`/spaces/${encodeURIComponent(id)}`)
};
