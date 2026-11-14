package mcp

import (
	mcpkit "github.com/FacileStudio/tronc/mcp"
	"github.com/go-chi/chi/v5"
)

// RegisterRoutes mounts the MCP endpoint at the resource's path and its RFC 9728
// metadata on router, the root router outside /api. A nil service mounts nothing.
func (s *Service) RegisterRoutes(router chi.Router) {
	if s != nil {
		s.server.Mount(router)
	}
}

// RegisterPendingRoutes mounts the confirm page's API under /mcp/pending on router, the
// /api group behind RequireAuth. A nil service, as in the docs test, still mounts the
// routes.
func (s *Service) RegisterPendingRoutes(router chi.Router) {
	s.pending().Mount(router)
}

// pending is the confirm page, nil on a nil service.
func (s *Service) pending() *mcpkit.PendingPage[PendingItem] {
	if s == nil {
		return nil
	}
	return s.page
}
