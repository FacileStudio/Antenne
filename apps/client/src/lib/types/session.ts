export interface SessionPayload {
	authenticated: boolean;
	passwordRequired: boolean;
	locale?: string;
	username: string;
	email: string;
	name?: string;
	role?: string;
	avatarUrl?: string;
}

/**
 * GET /api/auth/config. The first two keys are porte's and answer to the
 * deployment's configuration alone; the rest are Antenne's own.
 *
 * journal is absent unless JOURNAL_BROWSER_URL and JOURNAL_BROWSER_KEY are both
 * set, which is how hooks.client.ts tells "reporting is off" from "the
 * deployment is broken". The key is public by design: what protects the instance
 * is Journal's origin allowlist, its rate limit and its daily quota.
 */
export interface AuthConfigPayload {
	sso_only: boolean;
	oidc_enabled: boolean;
	password_required: boolean;
	username: string;
	email: string;
	journal?: { url: string; key: string };
}
