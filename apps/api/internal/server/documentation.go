package server

import "github.com/FacileStudio/tronc/apiref"

// SettingsDocumentation covers the configuration and action endpoints.
var SettingsDocumentation = apiref.Module{
	Name: "Settings",
	Description: "The whole configuration as one document: providers, delivery targets, system " +
		"events and the pool settings. Stored secrets read as ••••••••; sending that marker back keeps " +
		"the stored value, anything else replaces it, and an empty string clears it.",
	Routes: []apiref.Route{
		{
			Method:       "GET",
			Path:         "/api/settings",
			Summary:      "Read the settings",
			Description:  "Returns the redacted settings, the runtime status, and whether a usable Matrix target exists.",
			Auth:         "session",
			ResponseBody: "SettingsResponse",
		},
		{
			Method:       "PUT",
			Path:         "/api/settings",
			Summary:      "Replace the settings",
			Description:  "Replaces the whole document. A rejected save leaves the current settings untouched. The listener port takes effect on the next restart; everything else re-applies live.",
			Auth:         "session",
			RequestBody:  "AppSettings",
			ResponseBody: "SettingsResponse",
			Errors: []apiref.Error{
				{Status: 400, Code: "invalid_argument", Description: "a non-positive timeout, an unrooted webhook path, two enabled webhook providers on one path, an unknown delivery target type, or an invalid listener port"},
			},
		},
		{
			Method:       "POST",
			Path:         "/api/test-alert",
			Summary:      "Send a test alert",
			Description:  "Goes through the whole pipeline, routing rules included, so it stays silent unless a target lists the reserved provider id __antenne_system__.",
			Auth:         "session",
			ResponseBody: "OkResponse",
			Errors:       []apiref.Error{{Status: 400, Code: "invalid_argument", Description: "no delivery target is configured"}},
		},
		{
			Method:       "POST",
			Path:         "/api/test-delivery-target",
			Summary:      "Test one delivery target",
			Description:  "Sends straight to one target, bypassing the routing rules. This is the one that answers \"does this target work at all\".",
			Auth:         "session",
			RequestBody:  "TargetTestRequest",
			ResponseBody: "OkResponse",
			Errors:       []apiref.Error{{Status: 400, Code: "invalid_argument", Description: "targetId is required, unknown, or the send failed"}},
		},
		{
			Method:       "POST",
			Path:         "/api/test-provider-connection",
			Summary:      "Test a provider connection",
			Description:  "IMAP only: connects, logs in and selects INBOX. Every other provider type is rejected.",
			Auth:         "session",
			RequestBody:  "ProviderTestRequest",
			ResponseBody: "OkResponse",
			Errors:       []apiref.Error{{Status: 400, Code: "invalid_argument", Description: "providerId is required, unknown, not an IMAP provider, or the connection failed"}},
		},
		{
			Method:  "GET",
			Path:    "/api/delivery-health",
			Summary: "Read every target's health",
			Description: "One row per configured target, so a dashboard renders the whole state in one call. " +
				"State is one of unknown, ok, pending, down, recovering, flapping, stale or probe_error — unknown " +
				"means never measured or not measurable, and is never the same as ok. probe_error means Antenne's " +
				"own check could not run, not that the target refused. recovering means the target is answering " +
				"again while the outage stays open for the dwell time, so it is deliverable and not down. " +
				"detail is the far side's own words, verbatim. " +
				"anyDeliverable is false when no enabled target can currently receive anything, and " +
				"standingAlerts are the notices that outage swallowed — the only local record that " +
				"nobody was told.",
			Auth:         "session",
			ResponseBody: "DeliveryHealthPage",
		},
		{
			Method:  "POST",
			Path:    "/api/delivery-health/{targetId}/check",
			Summary: "Check one target now",
			Description: "Probes one target on demand. A handshake the far side declined answers 200 with ok:false " +
				"and its verbatim error, because the check itself worked — a non-2xx here means the request, the " +
				"session or this endpoint's own rate limit, never the target's verdict. Ten checks per client per " +
				"minute, on a budget shared with no other endpoint; over it, 429 carries Retry-After. " +
				"retryAfterSeconds in the body is the far side's own ask and is a different number. " +
				"state is where the target stands after the policy saw this observation, which is not the " +
				"observation itself: one failure inside the hysteresis threshold leaves a target pending.",
			Auth:         "session",
			PathParams:   []apiref.Field{{Name: "targetId", Type: "string", Description: "the delivery target id"}},
			ResponseBody: "TargetCheckResult",
			Errors: []apiref.Error{
				{Status: 404, Code: "not_found", Description: "no such delivery target"},
				{Status: 429, Code: "rate_limited", Description: "too many checks from this client; Retry-After says how long to wait"},
				{Status: 500, Code: "internal", Description: "the check could not run at all"},
			},
		},
	},
}

// EventsDocumentation covers the activity log.
var EventsDocumentation = apiref.Module{
	Name: "Events",
	Description: "The activity log: every event Antenne handled, what it was delivered to, and whether " +
		"that worked. The log keeps the most recent 5000 entries.",
	Routes: []apiref.Route{
		{
			Method:       "GET",
			Path:         "/api/events",
			Summary:      "List events",
			Description:  "Newest first. Query parameters: limit (default 50), offset, q for full-text search, and source to filter by webhook, website, imap, rss, system or test. The returned stats describe the whole filtered set, not the page.",
			Auth:         "session",
			ResponseBody: "EventPage",
		},
		{
			Method:       "GET",
			Path:         "/api/events/provider/{providerId}",
			Summary:      "List one provider's events",
			Auth:         "session",
			PathParams:   []apiref.Field{{Name: "providerId", Type: "string", Description: "the provider id"}},
			ResponseBody: "EventPage",
		},
		{
			Method:       "GET",
			Path:         "/api/events/target/{targetId}",
			Summary:      "List one delivery target's events",
			Description:  "Every event with a delivery attempt against this target, successful or not.",
			Auth:         "session",
			PathParams:   []apiref.Field{{Name: "targetId", Type: "string", Description: "the delivery target id"}},
			ResponseBody: "EventPage",
		},
		{
			Method:      "GET",
			Path:        "/api/events/stream",
			Summary:     "Stream new events",
			Description: "Server-sent events, one data: frame per new entry. Unfiltered: a client applies its own filters.",
			Auth:        "session",
		},
		{
			Method:       "POST",
			Path:         "/api/events/{eventId}/replay",
			Summary:      "Replay an event",
			Description:  "Re-sends a logged envelope to one target, bypassing the routing rules. Only events that carry an envelope can be replayed.",
			Auth:         "session",
			PathParams:   []apiref.Field{{Name: "eventId", Type: "string", Description: "the logged event id"}},
			RequestBody:  "TargetTestRequest",
			ResponseBody: "OkResponse",
			Errors: []apiref.Error{
				{Status: 400, Code: "invalid_argument", Description: "targetId is required, the event has no envelope, or the send failed"},
			},
		},
	},
}

// WebhooksDocumentation covers the third-party intake.
var WebhooksDocumentation = apiref.Module{
	Name: "Webhooks",
	Description: "Third-party receivers. The path and the secret header are per-provider settings, " +
		"so this one route stands for every configured provider. These paths stay at the root: they " +
		"are registered in Dokploy, DocuSeal, the git forges and Vikunja.",
	Routes: []apiref.Route{
		{
			Method:  "POST",
			Path:    "/webhook/{provider}",
			Summary: "Receive a webhook",
			Description: "The body is read once under a 4 MB cap and kept raw, because signature " +
				"verification hashes exactly what the sender signed; authentication runs before the " +
				"JSON is parsed. A recognised event that is not worth an alert — a bare heartbeat — is " +
				"accepted, logged and not delivered. Ardoise, Vision, Plume and Nuage verify an " +
				"HMAC-SHA256 of the body prefixed with sha256=; Opus verifies the bare hex digest; the " +
				"rest compare the configured header in constant time. An empty secret accepts " +
				"unauthenticated calls on that path.",
			Auth:         "provider secret or signature",
			PathParams:   []apiref.Field{{Name: "provider", Type: "string", Description: "the configured path segment, e.g. dokploy"}},
			ResponseBody: "OkResponse",
			Errors: []apiref.Error{
				{Status: 400, Code: "invalid_argument", Description: "the body is not JSON"},
				{Status: 401, Code: "unauthenticated", Description: "the secret or signature did not verify"},
				{Status: 404, Code: "not_found", Description: "no enabled provider is configured on that path"},
			},
		},
	},
}
