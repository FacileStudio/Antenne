package auth

import "github.com/FacileStudio/tronc/apiref"

// Documentation is this module's entry in the reference at /docs. A route
// absent from it fails TestEveryRouteIsDocumented.
var Documentation = apiref.Module{
	Name:        "Auth",
	Description: "Dashboard sessions. Antenne owns its users; there is no OIDC yet.",
	Routes: []apiref.Route{
		{
			Method:       "POST",
			Path:         "/api/login",
			Summary:      "Sign in",
			Description:  "Exchanges the admin password for a session cookie. Answers ok without doing anything when no admin password is configured. Failed attempts are rate-limited per client IP: five within ten minutes blocks that IP for ten minutes.",
			Auth:         "none",
			RequestBody:  "LoginRequest",
			ResponseBody: "OkResponse",
			Errors: []apiref.Error{
				{Status: 400, Code: "invalid_argument", Description: "the body is not JSON"},
				{Status: 401, Code: "unauthenticated", Description: "wrong password"},
				{Status: 429, Code: "rate_limited", Description: "too many attempts from this address"},
			},
		},
		{
			Method:       "POST",
			Path:         "/api/logout",
			Summary:      "Sign out",
			Description:  "Revokes the session this request carries and expires the cookie.",
			Auth:         "none",
			ResponseBody: "OkResponse",
		},
		{
			Method:       "GET",
			Path:         "/api/session/public",
			Summary:      "Read the public session state",
			Description:  "Whether a password is required at all, for the login page. Never reports an authenticated caller.",
			Auth:         "none",
			ResponseBody: "SessionResponse",
		},
		{
			Method:       "GET",
			Path:         "/api/session",
			Summary:      "Read the current session",
			Description:  "The authenticated caller and the login label the dashboard shows.",
			Auth:         "session",
			ResponseBody: "SessionResponse",
			Errors: []apiref.Error{
				{Status: 401, Code: "unauthenticated", Description: "no valid session cookie"},
			},
		},
	},
}
