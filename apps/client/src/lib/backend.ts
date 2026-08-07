import { goto } from '$app/navigation';
import type {
	AppSettings,
	EventEntry,
	EventStats,
	BusStats,
	SessionPayload,
	SettingsPayload,
	Space,
	SpaceMember,
	SpaceRole
} from './types';

/**
 * ApiError carries the API's own error code alongside the message, so a caller
 * can branch on `conflict` or `unauthenticated` without parsing prose.
 */
export class ApiError extends Error {
	readonly status: number;
	readonly code: string;

	constructor(message: string, status: number, code: string) {
		super(message);
		this.name = 'ApiError';
		this.status = status;
		this.code = code;
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
function parseError(body: string, status: number): ApiError {
	try {
		const payload = JSON.parse(body) as {
			error?: { code?: string; message?: string } | string;
			message?: string;
		};
		if (typeof payload.error === 'string') {
			return new ApiError(payload.error, status, 'unknown');
		}
		const message = payload.error?.message ?? payload.message;
		if (message) {
			return new ApiError(message, status, payload.error?.code ?? 'unknown');
		}
	} catch {
		// fall through to the status-only message
	}
	return new ApiError(`HTTP ${status}`, status, 'unknown');
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
	const response = await fetch(`/api${path}`, {
		...init,
		headers: { 'Content-Type': 'application/json', ...(init?.headers ?? {}) }
	});

	const body = await response.text();

	if (!response.ok) {
		throw parseError(body, response.status);
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

export async function redirectToLogin() {
	await goto('/login', { replaceState: true });
}

export const api = {
	session: () => request<SessionPayload>('/session'),
	publicSession: () => request<SessionPayload>('/session/public'),

	login: (password: string) =>
		request<{ ok: boolean }>('/login', { method: 'POST', body: JSON.stringify({ password }) }),
	logout: () => request<{ ok: boolean }>('/logout', { method: 'POST', body: '{}' }),

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

	spaces: () => request<{ spaces: Space[] }>('/spaces'),
	space: (id: string) =>
		request<{ space: Space; members: SpaceMember[] }>(`/spaces/${encodeURIComponent(id)}`),
	createSpace: (name: string, description: string) =>
		request<{ space: Space }>('/spaces', {
			method: 'POST',
			body: JSON.stringify({ name, description })
		}),
	updateSpace: (id: string, updates: { name?: string; description?: string }) =>
		request<{ space: Space }>(`/spaces/${encodeURIComponent(id)}`, {
			method: 'PUT',
			body: JSON.stringify(updates)
		}),
	deleteSpace: (id: string) =>
		request<{ ok: boolean }>(`/spaces/${encodeURIComponent(id)}`, { method: 'DELETE' }),

	members: (spaceId: string) =>
		request<{ members: SpaceMember[] }>(`/spaces/${encodeURIComponent(spaceId)}/members`),
	addMember: (spaceId: string, userId: string, role: SpaceRole) =>
		request<{ member: SpaceMember }>(`/spaces/${encodeURIComponent(spaceId)}/members`, {
			method: 'POST',
			body: JSON.stringify({ userId, role })
		}),
	updateMember: (spaceId: string, userId: string, role: SpaceRole) =>
		request<{ member: SpaceMember }>(
			`/spaces/${encodeURIComponent(spaceId)}/members/${encodeURIComponent(userId)}`,
			{ method: 'PUT', body: JSON.stringify({ role }) }
		),
	removeMember: (spaceId: string, userId: string) =>
		request<{ ok: boolean }>(
			`/spaces/${encodeURIComponent(spaceId)}/members/${encodeURIComponent(userId)}`,
			{ method: 'DELETE' }
		)
};
