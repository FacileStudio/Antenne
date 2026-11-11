package mcp

// eventsInput narrows antenne_list_events.
type eventsInput struct {
	Query          string `json:"query,omitempty" jsonschema:"words that must all appear in the event, its provider, its subject or its delivery targets"`
	Source         string `json:"source,omitempty" jsonschema:"only events from this source: webhook, website, imap, rss, test or system"`
	ProviderID     string `json:"provider_id,omitempty" jsonschema:"only events of this provider, an id from antenne_list_providers"`
	TargetID       string `json:"target_id,omitempty" jsonschema:"only events delivered to this target, an id from antenne_list_targets"`
	Limit          int    `json:"limit,omitempty" jsonschema:"events per page, 1 to 50, default 20"`
	Cursor         string `json:"cursor,omitempty" jsonschema:"next_cursor from the previous page"`
	ResponseFormat string `json:"response_format,omitempty" jsonschema:"concise (default) or detailed, which adds the subject, the actor and the full message"`
}

// eventInput names one event.
type eventInput struct {
	EventID string `json:"event_id" jsonschema:"the event id from antenne_list_events"`
}

// emptyInput is the argument of a tool that takes none.
type emptyInput struct{}

// targetInput names one delivery target.
type targetInput struct {
	TargetID string `json:"target_id" jsonschema:"the target id from antenne_list_targets"`
}

// providerInput names one provider.
type providerInput struct {
	ProviderID string `json:"provider_id" jsonschema:"the provider id from antenne_list_providers"`
}

// enabledInput switches one provider or target.
type enabledInput struct {
	Kind    string `json:"kind" jsonschema:"provider or target"`
	ID      string `json:"id" jsonschema:"the provider or target id"`
	Enabled bool   `json:"enabled" jsonschema:"true to turn it on, false to turn it off"`
}

// replayInput names the event to send again and where.
type replayInput struct {
	EventID  string `json:"event_id" jsonschema:"the event id from antenne_list_events"`
	TargetID string `json:"target_id" jsonschema:"the target id from antenne_list_targets"`
}

// eventsOutput is one page of the activity log.
type eventsOutput struct {
	Events     []eventOut `json:"events"`
	Total      int64      `json:"total"`
	Stats      statsOut   `json:"stats"`
	NextCursor string     `json:"next_cursor,omitempty"`
}

// statsOut sums up the events that match the filter, every page included.
type statsOut struct {
	Events      int     `json:"events"`
	LastEventAt *string `json:"last_event_at,omitempty"`
	Delivered   int     `json:"delivered"`
	Failed      int     `json:"failed_deliveries"`
}

// eventOut is one logged event.
type eventOut struct {
	ID           string       `json:"id"`
	At           string       `json:"at"`
	Source       string       `json:"source"`
	ProviderID   string       `json:"provider_id,omitempty"`
	Provider     string       `json:"provider"`
	ProviderType string       `json:"provider_type"`
	Event        string       `json:"event"`
	Title        string       `json:"title,omitempty"`
	Message      string       `json:"message,omitempty"`
	URL          string       `json:"url,omitempty"`
	Deliveries   []attemptOut `json:"deliveries"`
	Detail       *eventDetail `json:"detail,omitempty"`
}

// eventDetail is what the detailed format adds to an event.
type eventDetail struct {
	Type    string `json:"type,omitempty"`
	Subject string `json:"subject,omitempty"`
	Actor   string `json:"actor,omitempty"`
}

// attemptOut is one delivery of an event to a target.
type attemptOut struct {
	TargetID string `json:"target_id"`
	Target   string `json:"target"`
	Status   string `json:"status"`
	Error    string `json:"error,omitempty"`
}

// eventFull is antenne_get_event: the event with its envelope payload.
type eventFull struct {
	eventOut
	Normalized any    `json:"normalized,omitempty"`
	Raw        string `json:"raw,omitempty"`
	Truncated  bool   `json:"truncated,omitempty"`
}

// providersOutput lists the providers.
type providersOutput struct {
	Providers []providerOut `json:"providers"`
}

// providerOut is one provider, without its credentials.
type providerOut struct {
	ID      string   `json:"id"`
	Type    string   `json:"type"`
	Name    string   `json:"name"`
	Enabled bool     `json:"enabled"`
	Tags    []string `json:"tags"`
	Watches string   `json:"watches,omitempty"`
}

// targetsOutput lists the delivery targets with their health.
type targetsOutput struct {
	Targets        []targetOut   `json:"targets"`
	AnyDeliverable bool          `json:"any_deliverable"`
	StandingAlerts []standingOut `json:"standing_alerts"`
}

// targetOut is one delivery target, without its credentials or address.
type targetOut struct {
	ID           string   `json:"id"`
	Type         string   `json:"type"`
	Name         string   `json:"name"`
	Enabled      bool     `json:"enabled"`
	ProviderIDs  []string `json:"provider_ids"`
	ProviderTags []string `json:"provider_tags"`
	State        string   `json:"state"`
	Deliverable  bool     `json:"deliverable"`
	Detail       string   `json:"detail,omitempty"`
	DownSince    string   `json:"down_since,omitempty"`
	LastSuccess  string   `json:"last_success_at,omitempty"`
}

// standingOut is an alert no target could carry.
type standingOut struct {
	TargetID string `json:"target_id"`
	Target   string `json:"target"`
	Title    string `json:"title"`
	Detail   string `json:"detail"`
	Since    string `json:"since"`
}

// poolOutput is the live bus, without any token.
type poolOutput struct {
	Connections int       `json:"connections"`
	Apps        []poolApp `json:"apps"`
}

// poolApp is one app connected to the bus.
type poolApp struct {
	App      string   `json:"app"`
	AppID    string   `json:"app_id"`
	Channels []string `json:"listens"`
	Emits    []string `json:"emits"`
}

// checkOutput is one on-demand probe.
type checkOutput struct {
	TargetID          string `json:"target_id"`
	OK                bool   `json:"ok"`
	Kind              string `json:"kind"`
	Detail            string `json:"detail,omitempty"`
	State             string `json:"state"`
	RetryAfterSeconds int    `json:"retry_after_seconds,omitempty"`
}

// sendOutput is a test sent to a target or a provider connection tried.
type sendOutput struct {
	Status string `json:"status"`
	ID     string `json:"id"`
	Error  string `json:"error,omitempty"`
}

// enabledOutput is a provider or target after the switch.
type enabledOutput struct {
	Kind    string `json:"kind"`
	ID      string `json:"id"`
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
}

// replayOutput is where a replay stands.
type replayOutput struct {
	Status     string `json:"status"`
	EventID    string `json:"event_id,omitempty"`
	TargetID   string `json:"target_id,omitempty"`
	ConfirmURL string `json:"confirm_url,omitempty"`
	Message    string `json:"message,omitempty"`
}
