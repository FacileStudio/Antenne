export interface SessionPayload {
	authenticated: boolean;
	passwordRequired: boolean;
	username: string;
	email: string;
}

/**
 * GET /api/auth/config. The first two keys are porte's and answer to the
 * deployment's configuration alone; the last three are Antenne's own.
 */
export interface AuthConfigPayload {
	sso_only: boolean;
	oidc_enabled: boolean;
	password_required: boolean;
	username: string;
	email: string;
}
