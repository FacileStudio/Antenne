package pool

import "github.com/FacileStudio/tronc/apiref"

// Documentation is this module's entry in the reference at /docs.
//
// The root-level and /api-prefixed forms of register and ws are both listed,
// because both are real routes: pool clients pin whichever they were given, and
// dropping either would break them.
var Documentation = apiref.Module{
	Name: "Pool",
	Description: "The Antenne Pool: a WebSocket bus with a durable offset-based queue that other " +
		"Facile apps register with and sync through. These endpoints authenticate with the pool " +
		"secret and a connection token, not with a dashboard session.",
	Routes: []apiref.Route{
		{
			Method:       "POST",
			Path:         "/antenne/register",
			Summary:      "Register an app",
			Description:  "Exchanges the shared secret for a connection token. Re-registering an app keeps its existing token, so a restarting client does not invalidate the token another instance is using.",
			Auth:         "pool secret",
			RequestBody:  "PoolRegistrationRequest",
			ResponseBody: "PoolRegistrationResponse",
			Errors: []apiref.Error{
				{Status: 400, Code: "invalid_argument", Description: "app is required, or the body is not JSON"},
				{Status: 403, Code: "permission_denied", Description: "the secret did not match"},
			},
		},
		{
			Method:       "POST",
			Path:         "/api/antenne/register",
			Summary:      "Register an app (prefixed)",
			Description:  "Identical to POST /pool/register. Both forms exist because clients pin whichever they were given.",
			Auth:         "pool secret",
			RequestBody:  "PoolRegistrationRequest",
			ResponseBody: "PoolRegistrationResponse",
		},
		{
			Method:      "GET",
			Path:        "/antenne/ws",
			Summary:     "Open the pool socket",
			Description: "Upgrades to WebSocket. The token goes in the query string. One live connection per app: a second replaces the first with close code 1000 'replaced by new connection'. See docs/api.md for the frame protocol.",
			Auth:        "pool token",
			Errors: []apiref.Error{
				{Status: 401, Code: "unauthenticated", Description: "no token in the query string"},
				{Status: 403, Code: "permission_denied", Description: "the token does not resolve to a registered app"},
			},
		},
		{
			Method:      "GET",
			Path:        "/api/antenne/ws",
			Summary:     "Open the pool socket (prefixed)",
			Description: "Identical to GET /pool/ws.",
			Auth:        "pool token",
		},
		{
			Method:       "GET",
			Path:         "/api/antenne/stats",
			Summary:      "Read the live pool state",
			Description:  "The boot epoch, the live connection count, and every connected app with the channels it subscribed to.",
			Auth:         "session",
			ResponseBody: "PoolStats",
			Errors:       []apiref.Error{{Status: 401, Code: "unauthenticated", Description: "no valid session cookie"}},
		},
	},
}
