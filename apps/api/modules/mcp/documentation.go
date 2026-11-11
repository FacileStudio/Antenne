package mcp

import (
	"github.com/FacileStudio/tronc/apiref"
	mcpkit "github.com/FacileStudio/tronc/mcp"
)

// Documentation describes the confirm page routes for the API reference. The MCP endpoint
// itself is outside /api: POST /mcp speaks the Model Context Protocol to any MCP client
// holding a Registre access token for this server, and
// /.well-known/oauth-protected-resource/mcp describes how to get one.
func Documentation() apiref.Module {
	return apiref.Module{
		Name: "MCP",
		Description: "Event replays MCP clients asked for, waiting for the person to confirm them. A client that " +
			"cannot ask the person itself hands them a link to the pending page, which uses these routes. The MCP " +
			"endpoint is POST /mcp, outside /api.",
		Routes: pendingRoutes(),
	}
}

// pendingRoutes documents the three confirm page routes, under /api where they are
// mounted.
func pendingRoutes() []apiref.Route {
	routes := mcpkit.PendingRoutes[PendingItem](mcpkit.PendingDoc{
		Auth: "session",
		List: "Open actions, newest first, with their event and target as they are now. state is ready, changed " +
			"(the event or the target was edited after the request, so it cannot be confirmed) or missing.",
		Confirm: "Runs the action once and sends the event to the target. A target edited since the request is " +
			"refused with 412 and the action closes.",
		Dismiss: "Closes the action without running it.",
		Changed: "The event or the target changed after the request.",
	})
	for i := range routes {
		routes[i].Path = "/api" + routes[i].Path
	}
	return routes
}
