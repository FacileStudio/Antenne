import type {
	AppSettings,
	DeliveryHealthPage,
	EventEntry,
	EventStats,
	BusStats,
	PendingAction,
	TargetCheckResult,
	SessionPayload,
	AuthConfigPayload,
	SettingsPayload,
	Space,
	SpaceMember
} from './types';
import { ApiError, authConfig, createApi, isUnauthorized, redirectToLogin } from '@facile/client';

const request = createApi();

export { ApiError, isUnauthorized as isUnauthenticated, redirectToLogin };

export function isRateLimited(error: unknown): boolean {
	return error instanceof ApiError && error.status === 429;
}

export const api = {
	session: () => request<SessionPayload>('/session'),

	/**
	 * What this instance accepts, before holding any credential. porte owns the
	 * route and serves sso_only and oidc_enabled; password_required, username
	 * and email are Antenne's own additions to it.
	 */
	authConfig: () => authConfig<AuthConfigPayload>(request),

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

	/** Replays an AI client asked for, waiting for this person to confirm them. */
	pending: () => request<{ actions: PendingAction[] }>('/mcp/pending'),
	confirmPending: (id: string) =>
		request<{ status: string }>(`/mcp/pending/${encodeURIComponent(id)}/confirm`, { method: 'POST', body: '{}' }),
	dismissPending: (id: string) =>
		request<{ status: string }>(`/mcp/pending/${encodeURIComponent(id)}/dismiss`, { method: 'POST', body: '{}' }),

	spaces: () => request<{ spaces: Space[]; manageUrl: string }>('/spaces'),
	space: (id: string) =>
		request<{ space: Space; members: SpaceMember[] }>(`/spaces/${encodeURIComponent(id)}`)
};
