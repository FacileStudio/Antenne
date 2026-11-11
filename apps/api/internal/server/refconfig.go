package server

import (
	"github.com/FacileStudio/Antenne/apps/api/modules/auth"
	"github.com/FacileStudio/Antenne/apps/api/modules/bus"
	"github.com/FacileStudio/Antenne/apps/api/modules/spaces"
	"github.com/FacileStudio/tronc/apiref"
)

// RefConfig describes the API reference served at /docs.
//
// Servers is "/" rather than "/api" because Antenne's surface genuinely is not all
// under one prefix: the webhook receivers and the pool endpoints live at the
// root, where the systems that call them have them registered. Every route is
// therefore documented by its full path. extra are the modules mounted only when
// configured, such as the MCP server's confirm page.
func RefConfig(extra ...apiref.Module) apiref.Config {
	return apiref.Config{
		Title:       "Antenne API",
		Description: "Alert aggregation and the Facile Suite's event bus. Configuration, the activity log, the third-party webhook intake and the Antenne Pool.",
		Servers:     []string{"/"},
		Registry: apiref.Registry{Modules: append([]apiref.Module{
			auth.Documentation(),
			SettingsDocumentation(),
			EventsDocumentation(),
			WebhooksDocumentation(),
			bus.Documentation(),
			spaces.Documentation(),
		}, extra...)},
	}
}
