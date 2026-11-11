package mcp

import (
	mcpkit "github.com/FacileStudio/tronc/mcp"
)

// list is the curated tool set. The event log is open world: providers relay content
// from third parties. Probes and test sends reach the far side of a target. Replaying an
// event is the sensitive tool.
func (t *toolset) list() []mcpkit.Tool {
	return append(append(t.reads(), t.writes()...), t.sensitive()...)
}

// reads are the tools that change nothing.
func (t *toolset) reads() []mcpkit.Tool {
	return append(t.logReads(), t.setupReads()...)
}

// logReads are the activity log tools.
func (t *toolset) logReads() []mcpkit.Tool {
	return []mcpkit.Tool{
		mcpkit.NewTool(mcpkit.ToolSpec{
			Name: toolEvents, Title: "List events",
			Description: "List the activity log, newest first: each event with its source, provider, title, a short " +
				"plain-text message and where it was delivered, plus the total and delivery counts for the filter. " +
				"Pass query, source, provider_id or target_id to narrow it, and next_cursor back as cursor for the " +
				"next page. The log keeps the last 5000 events.",
			Annotations: &mcpkit.Annotations{ReadOnly: true, Idempotent: true, OpenWorld: true},
		}, t.listEvents),
		mcpkit.NewTool(mcpkit.ToolSpec{
			Name: toolEvent, Title: "Read an event",
			Description: "Read one event in full: its message, subject, actor, deliveries, the normalized payload " +
				"and the raw body the source sent, cut to about 20 KB.",
			Annotations: &mcpkit.Annotations{ReadOnly: true, Idempotent: true, OpenWorld: true},
		}, t.getEvent),
	}
}

// setupReads are the tools that read the providers, the targets and the bus.
func (t *toolset) setupReads() []mcpkit.Tool {
	return []mcpkit.Tool{
		mcpkit.NewTool(mcpkit.ToolSpec{
			Name: toolProviders, Title: "List providers",
			Description: "List the providers Antenne watches (suite and third-party webhooks, websites, IMAP " +
				"inboxes, RSS feeds): id, type, name, on or off, tags, and what each watches. Credentials are " +
				"never returned.",
			Annotations: &mcpkit.Annotations{ReadOnly: true, Idempotent: true},
		}, t.listProviders),
		mcpkit.NewTool(mcpkit.ToolSpec{
			Name: toolTargets, Title: "List delivery targets",
			Description: "List the delivery targets with their routing (the provider ids and tags each receives) " +
				"and their health: state (ok, pending, down, recovering, unknown), whether alerts can reach it, the far side's " +
				"last words, plus any alert no target could carry. Addresses and credentials are never returned.",
			Annotations: &mcpkit.Annotations{ReadOnly: true, Idempotent: true},
		}, t.listTargets),
		mcpkit.NewTool(mcpkit.ToolSpec{
			Name: toolPool, Title: "Bus status",
			Description: "Show the suite apps connected to the Antenne bus right now, with the channels each " +
				"listens to and emits. Tokens and the pool secret are never returned.",
			Annotations: &mcpkit.Annotations{ReadOnly: true, Idempotent: true},
		}, t.poolStatus),
		mcpkit.NewTool(mcpkit.ToolSpec{
			Name: toolTestSource, Title: "Test an IMAP provider",
			Description: "Log in to an IMAP provider's mailbox once and report ok or failed with the server's " +
				"error. Only IMAP providers have a connection to test.",
			Annotations: &mcpkit.Annotations{ReadOnly: true, Idempotent: true, OpenWorld: true},
		}, t.testProvider),
	}
}

// writes are the tools that reach a target or change the configuration.
func (t *toolset) writes() []mcpkit.Tool {
	return []mcpkit.Tool{
		mcpkit.NewTool(mcpkit.ToolSpec{
			Name: toolCheck, Title: "Check a target",
			Description: "Probe one delivery target now, without sending a message, and record the result in its " +
				"health. Returns ok, the kind of answer, the far side's words and the state after the check. " +
				"Limited to a burst of 5, then one every 12 seconds.",
			Annotations: &mcpkit.Annotations{Idempotent: true, OpenWorld: true},
		}, t.checkTarget),
		mcpkit.NewTool(mcpkit.ToolSpec{
			Name: toolTest, Title: "Send a test message",
			Description: "Send Antenne's fixed test message to one delivery target, past the routing rules, to see " +
				"whether it works. Returns status ok, or failed with the far side's error.",
			Annotations: &mcpkit.Annotations{OpenWorld: true},
		}, t.testTarget),
		mcpkit.NewTool(mcpkit.ToolSpec{
			Name: toolEnabled, Title: "Turn a provider or target on or off",
			Description: "Turn one provider (kind provider) or delivery target (kind target) on or off, as the " +
				"dashboard switch does. A provider that is off stops watching; a target that is off receives " +
				"nothing until it is turned on again.",
			Annotations: &mcpkit.Annotations{Destructive: true, Idempotent: true},
		}, t.setEnabled),
	}
}

// sensitive are the tools the user confirms.
func (t *toolset) sensitive() []mcpkit.Tool {
	return []mcpkit.Tool{
		mcpkit.NewTool(mcpkit.ToolSpec{
			Name: toolReplay, Title: "Replay an event to a target",
			Description: "Send one logged event again to one delivery target, which may be one the routing never " +
				"sent it to. The user always confirms: in your client when it can ask, otherwise the result has " +
				"status pending_confirmation and a confirm_url to give the user. Nothing is sent before they confirm.",
			Annotations: &mcpkit.Annotations{OpenWorld: true},
		}, t.replayEvent),
	}
}
