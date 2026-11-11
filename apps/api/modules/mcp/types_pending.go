package mcp

import (
	mcpkit "github.com/FacileStudio/tronc/mcp"
)

// PendingItem is one action waiting on the confirm page.
type PendingItem struct {
	mcpkit.PendingItem
	Event  *PendingEvent  `json:"event,omitempty"`
	Target *PendingTarget `json:"target,omitempty"`
}

// PendingEvent is the logged event a replay would send, as it is now.
type PendingEvent struct {
	ID       string `json:"id"`
	At       string `json:"at"`
	Source   string `json:"source"`
	Provider string `json:"provider"`
	Event    string `json:"event"`
	Title    string `json:"title,omitempty"`
}

// PendingTarget is the delivery target a replay would send to, as it is now.
type PendingTarget struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Type  string `json:"type"`
	Label string `json:"label"`
}
