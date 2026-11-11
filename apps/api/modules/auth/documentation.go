package auth

import (
	"github.com/FacileStudio/porte"
	"github.com/FacileStudio/porte/kit"
	"github.com/FacileStudio/tronc/apiref"
)

type AuthConfigResponse struct {
	SSOOnly          bool   `json:"sso_only"`
	OIDCEnabled      bool   `json:"oidc_enabled"`
	PasswordRequired bool   `json:"password_required"`
	Username         string `json:"username,omitempty"`
	Email            string `json:"email,omitempty"`
}

func Documentation() apiref.Module {
	routes := passwordRoutes()
	routes = append(routes, oidcRoutes()...)
	routes = append(routes, sessionRoutes()...)
	routes = append(routes, invitationsRoute())
	return apiref.Module{
		Name:        "Auth",
		Description: "Sessions, on the suite's porte kit. One session model and two transports: an HttpOnly cookie for the dashboard, a bearer token for the CLI. Single sign-on against Authentik when OIDC_ISSUER is set, plus the bootstrap password login.",
		Routes:      routes,
	}
}

func passwordRoutes() []apiref.Route {
	return []apiref.Route{
		{
			Method:       "POST",
			Path:         "/api/auth/login",
			Summary:      "Sign in with the bootstrap password",
			Description:  "Exchanges the admin address and password for a session: the cookie is set and the bearer token is returned, so one call serves the dashboard and the CLI. Not registered at all once OIDC_ISSUER is set, nor when no admin password is configured. Refused attempts are throttled per client address: five within ten minutes blocks that address for ten minutes.",
			Auth:         "none",
			RequestBody:  porte.CredentialsRequest{},
			ResponseBody: porte.ExchangeResponse{},
			Errors: []apiref.Error{
				{Status: 400, Code: "invalid_argument", Description: "the body is not JSON, or the address is not an address"},
				{Status: 401, Code: "unauthenticated", Description: "wrong address or password — deliberately the same answer for both, and it costs the same time"},
				{Status: 429, Code: "rate_limited", Description: "too many refused attempts from this address"},
			},
		},
		{
			Method:       "POST",
			Path:         "/api/auth/logout",
			Summary:      "Sign out",
			Description:  "Revokes the session this request carries and expires the cookie. Idempotent: a stale cookie is cleared rather than refused, because the state that makes someone log out is the state where the credential is already dead. A caller presenting a cookie must still send X-Facile-CSRF.",
			Auth:         "session or bearer",
			ResponseBody: porte.LogoutResponse{},
		},
		{
			Method:       "GET",
			Path:         "/api/auth/config",
			Summary:      "Read what this instance accepts",
			Description:  "Whether single sign-on is available, whether it is mandatory, and whether a password form should be drawn at all. The login page reads this before showing anything.",
			Auth:         "none",
			ResponseBody: AuthConfigResponse{},
		},
	}
}

func oidcRoutes() []apiref.Route {
	return []apiref.Route{
		{
			Method:      "GET",
			Path:        "/api/auth/oidc",
			Summary:     "Start the single sign-on flow",
			Description: "Redirects to Authentik with PKCE, a nonce and a state cookie. A CLI asks for the one-time-code flow with ?flow=cli&port=N and should send its own ?cli_state nonce. Registered only when OIDC_ISSUER is set.",
			Auth:        "none",
		},
		{
			Method:      "GET",
			Path:        "/api/auth/oidc/callback",
			Summary:     "Complete the single sign-on flow",
			Description: "Verifies the token, matches the account on (provider, subject), upserts it and issues the session. This is a browser navigation, so both outcomes are a 302: read Location, not the status. A failure lands on the login page with the reason in an error query parameter.",
			Auth:        "none",
		},
		{
			Method:       "POST",
			Path:         "/api/auth/oidc/exchange",
			Summary:      "Trade a CLI login code for a token",
			Description:  "The one-time code issued to a CLI at the end of a browser login, exchanged for a bearer token. Single use, sixty seconds.",
			Auth:         "none",
			RequestBody:  porte.ExchangeRequest{},
			ResponseBody: porte.ExchangeResponse{},
			Errors: []apiref.Error{
				{Status: 400, Code: "invalid_argument", Description: "the code is unknown, expired, or already consumed"},
			},
		},
		{
			Method:       "POST",
			Path:         "/api/auth/sync-profile",
			Summary:      "Refresh the profile from the identity provider",
			Description:  "Pulls the name and address Authentik holds. Rate-limited to one refresh per account per five minutes; a call inside that window answers synced false rather than failing.",
			Auth:         "session or bearer",
			ResponseBody: porte.SyncProfileResponse{},
		},
		{
			Method:      "POST",
			Path:        "/api/auth/backchannel-logout",
			Summary:     "Accept an administrative logout from the identity provider",
			Description: "Authentik calls this when a session is terminated centrally — deactivating a user there drops them here within seconds. It is the only way this app learns of it, since the session it issued is opaque and long-lived.",
			Auth:        "signed logout token",
		},
	}
}

func sessionRoutes() []apiref.Route {
	return []apiref.Route{
		{
			Method:       "GET",
			Path:         "/api/session",
			Summary:      "Read the current session",
			Description:  "The authenticated caller, and the login label the dashboard shows.",
			Auth:         "session or bearer",
			ResponseBody: SessionResponse{},
			Errors: []apiref.Error{
				{Status: 401, Code: "unauthenticated", Description: "no valid session"},
			},
		},
	}
}

// invitationsRoute documents porte/kit's pending invitations route at its full
// path, because this reference documents every route from the root. It is
// mounted only when single sign-on and the directory credentials are set.
func invitationsRoute() apiref.Route {
	route := kit.InvitationsRoute()
	route.Path = "/api" + kit.InvitationsPath
	route.ResponseBody = kit.InvitationsResponse{}
	route.Description += " Mounted only when OIDC_ISSUER and OIDC_CLIENT_SECRET are set."
	return route
}
