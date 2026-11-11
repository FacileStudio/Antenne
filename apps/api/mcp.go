package main

import (
	"context"
	"database/sql"
	"log/slog"

	"github.com/FacileStudio/Antenne/apps/api/internal/env"
	"github.com/FacileStudio/Antenne/apps/api/internal/server"
	"github.com/FacileStudio/Antenne/apps/api/modules/mcp"
	"github.com/FacileStudio/Antenne/apps/api/modules/spaces"
	"github.com/FacileStudio/porte"
	"github.com/FacileStudio/porte/mcpauth"
	"github.com/FacileStudio/porte/oidc"
	mcpkit "github.com/FacileStudio/tronc/mcp"
)

// undocumented are the routes the API reference leaves out: the legacy /dashboard
// redirect, the version probe, and the MCP endpoint with its RFC 9728 metadata, which
// speak their own protocols.
var undocumented = []string{"/dashboard", "/version", "/mcp", "/.well-known"}

// newMCP builds the MCP server when single sign-on is on and a resource is set, and
// nothing otherwise. The verifier runs Registre discovery here, so a wrong issuer fails
// at boot like the OIDC kit does. A person's first MCP call creates their account through
// the OIDC kit, as a first web login would, so it is an administrator like every account.
func newMCP(ctx context.Context, cfg env.Config, sqlDB *sql.DB, kit *oidc.Kit, identities porte.IdentityStore, logger *slog.Logger, deps mcp.Deps) (server.MCP, error) {
	if cfg.MCPResource == "" || !kit.Enabled() {
		return nil, nil
	}
	verifier, err := mcpauth.Verifier(ctx, mcpauth.Config{
		Issuer:      cfg.Porte.Issuer,
		Resource:    cfg.MCPResource,
		ScopePrefix: "antenne",
		Identities:  identities,
		Provision:   true,
		Accounts:    kit,
		Logger:      logger,
	})
	if err != nil {
		return nil, err
	}
	service, err := mcp.NewService(deps, mcp.Config{
		Resource: cfg.MCPResource,
		Issuer:   cfg.Porte.Issuer,
		Version:  cfg.Version,
		Origins:  cfg.CORSAllowedOrigins,
		StateKey: cfg.MCPStateKey(),
		Verifier: verifier,
		Pending:  mcpkit.PostgresPending{DB: sqlDB},
		Logger:   logger,
	})
	if err != nil {
		return nil, err
	}
	logger.Info("mcp server on", slog.String("resource", cfg.MCPResource))
	return service, nil
}

// eraseAccountData deletes what an erased person owns, in the transaction that deletes
// their account: their space memberships and their MCP actions. The event log, the
// providers and the delivery targets belong to the instance and stay.
func eraseAccountData(ctx context.Context, tx *sql.Tx, userID int64) error {
	if err := spaces.EraseMemberships(ctx, tx, userID); err != nil {
		return err
	}
	return mcp.EraseAccount(ctx, tx, userID)
}
