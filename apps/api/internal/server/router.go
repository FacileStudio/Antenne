// Package server wires the HTTP surface: the dashboard API, the webhook
// intake, the pool endpoints and the SPA.
package server

import (
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/FacileStudio/tronc/apiref"
	apierrors "github.com/FacileStudio/tronc/errors"
	"github.com/FacileStudio/tronc/health"
	"github.com/FacileStudio/tronc/httpjson"
	"github.com/FacileStudio/tronc/httpx"
	"github.com/FacileStudio/tronc/middleware"
	"github.com/FacileStudio/tronc/spa"
	"github.com/go-chi/chi/v5"

	"github.com/FacileStudio/Antenne/apps/api/internal/app"
	"github.com/FacileStudio/Antenne/apps/api/internal/env"
	"github.com/FacileStudio/Antenne/apps/api/modules/auth"
	"github.com/FacileStudio/Antenne/apps/api/modules/bus"
	"github.com/FacileStudio/Antenne/apps/api/modules/spaces"
	"github.com/FacileStudio/porte/directory"
	"github.com/FacileStudio/porte/kit"
	"github.com/FacileStudio/porte/local"
	"github.com/FacileStudio/porte/oidc"
	"github.com/FacileStudio/porte/session"
)

// MCP is the remote MCP server: its endpoint and metadata at the root, its confirm
// page's API for a signed-in person, and that API's reference.
type MCP interface {
	RegisterRoutes(router chi.Router)
	RegisterPendingRoutes(router chi.Router)
	Documentation() apiref.Module
}

// BuildRouter is the composition root: wiring the health endpoint, the API
// reference, the three groups (public, auth-protected, sessioned) and the SPA
// catch-all. The caller passes the assembled dependencies and the router is the
// result, or the error porte/kit returned while mounting the sign-in routes. A nil
// mcpServer leaves /mcp unmounted.
func BuildRouter(
	cfg env.Config,
	log *slog.Logger,
	sqlDB *sql.DB,
	application *app.App,
	api *API,
	authService *auth.Service,
	sessions *session.Manager,
	oidcKit *oidc.Kit,
	passwords *local.Kit,
	spaceService *spaces.Service,
	busServer *bus.Server,
	identities kit.IdentityLister,
	mcpServer MCP,
) (chi.Router, error) {
	router := httpx.NewRouter(httpx.Config{
		// Behind Traefik and Cloudflare, RemoteAddr is only the
		// visitor if both are trusted: Traefik replaces the forwarded
		// chain rather than extending it, so the visitor survives in
		// Cf-Connecting-Ip alone. TRUSTED_PROXIES=private,cloudflare
		// fills all three.
		TrustedProxies: cfg.TrustedProxies,
		CDNProxies:     cfg.CDNProxies,
		CDNHeader:      cfg.CDNHeader,
		Logger:         log,
		Security:       securityConfig(cfg, log),
		CORS: middleware.CORSConfig{
			AllowedOrigins:   cfg.CORSAllowedOrigins,
			AllowCredentials: true,
		},
	})

	// Mounted at both the root and /api: an external monitor that only reaches
	// one of the two must not read the SPA shell as a green check.
	health.Mount(router, health.DB(sqlDB))
	router.Get("/version", func(w http.ResponseWriter, _ *http.Request) {
		httpjson.WriteJSON(w, http.StatusOK, map[string]string{"version": cfg.Version})
	})

	// The reference stays at the root, unauthenticated, next to the routes it
	// documents. It describes the public contract — the webhook intake and the
	// pool protocol other apps integrate against — and carries no data.
	apiref.Mount(router, RefConfig(mcpDocs(mcpServer)...))
	if mcpServer != nil {
		mcpServer.RegisterRoutes(router)
	}

	var mountErr, invitationsErr error

	router.Route("/api", func(protected chi.Router) {
		// Scoped to the /api subtree, so an unknown API path 404s as JSON
		// instead of falling through to the SPA catch-all. A machine client that
		// treats any 2xx as success would otherwise read a mistyped URL as a
		// successful write and discard the payload with no error anywhere.
		protected.NotFound(func(w http.ResponseWriter, r *http.Request) {
			httpjson.WriteError(w, apierrors.NotFound("not found"))
		})
		protected.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
			httpjson.WriteJSON(w, http.StatusMethodNotAllowed, map[string]any{
				"error": map[string]string{"code": "method_not_allowed", "message": "method not allowed"},
			})
		})

		// porte owns the credentials: the logout, the whole OIDC flow and
		// /auth/config come from the kits, mounted before the guard because
		// they are how a caller acquires the credential the guard demands.
		mountErr = kit.Mount(protected, kit.Config{Sessions: sessions, OIDC: oidcKit})

		// With an issuer configured the password route is not registered
		// rather than rejected, so there is no endpoint left to probe for an
		// account.
		if authService.PasswordRequired() {
			passwords.Mount(protected.With(auth.ThrottleLogins()))
		}

		busServer.RegisterRoutes(protected)
		spaceService.RegisterSync(protected)
		spaceService.RegisterDeletions(protected)

		protected.Group(func(sessioned chi.Router) {
			sessioned.Use(auth.RequireAuth(sessions, authService))
			authService.RegisterRoutes(sessioned)
			api.RegisterRoutes(sessioned)
			spaceService.RegisterRoutes(sessioned)
			busServer.RegisterStats(sessioned)
			invitationsErr = mountInvitations(sessioned, cfg, identities, log)
			if mcpServer != nil {
				mcpServer.RegisterPendingRoutes(sessioned)
			}
		})
	})

	// The webhook paths are registered in third-party systems — Dokploy,
	// DocuSeal, a git forge — and cannot be moved under /api without breaking
	// every one of them. Same for /pool/ws, which pool clients pin.
	webhookHandler := NewWebhookHandler(application, busServer, log)
	router.Post("/webhook/*", webhookHandler.ServeHTTP)
	busServer.RegisterRoutes(router)

	// The dashboard used to live under /dashboard; the SPA is at the root now,
	// so old bookmarks and the post-login redirect are moved rather than 404ed.
	router.Handle("/dashboard", http.RedirectHandler("/", http.StatusMovedPermanently))
	router.Get("/dashboard/*", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/"+strings.TrimPrefix(r.URL.Path, "/dashboard/"), http.StatusMovedPermanently)
	})

	if spa.Available(cfg.ClientDir) {
		router.NotFound(spa.Handler(spa.Config{Dir: cfg.ClientDir}).ServeHTTP)
	}

	return router, errors.Join(mountErr, invitationsErr)
}

// mcpDocs is the reference of the MCP server's routes, or nothing without one.
func mcpDocs(mcpServer MCP) []apiref.Module {
	if mcpServer == nil {
		return nil
	}
	return []apiref.Module{mcpServer.Documentation()}
}

// mountInvitations serves the user menu's pending Registre invitations. It
// needs the directory credentials the deletion receiver already uses, so an
// instance without single sign-on, or without a client secret, mounts nothing.
func mountInvitations(router chi.Router, cfg env.Config, identities kit.IdentityLister, log *slog.Logger) error {
	if cfg.Porte.Issuer == "" || cfg.Porte.ClientSecret == "" || identities == nil {
		return nil
	}
	return kit.MountInvitations(router, kit.InvitationsConfig{
		Directory: directory.Client{
			Issuer:       cfg.Porte.Issuer,
			ClientID:     cfg.Porte.ClientID,
			ClientSecret: cfg.Porte.ClientSecret,
		},
		Identities: identities,
		Logger:     log,
	})
}

// securityConfig extends tronc's default headers with what the dashboard loads.
// The built client carries two inline scripts, allowed by hash. iconify-icon
// fetches any glyph muse does not bundle from the Iconify API, and the Journal
// browser SDK reports to its own origin. The policy is still report-only in
// tronc v0.17, so a miss shows in the console and breaks nothing.
func securityConfig(cfg env.Config, log *slog.Logger) middleware.SecurityConfig {
	connect := []string{"https://api.iconify.design", "https://api.simplesvg.com", "https://api.unisvg.com"}
	if journal := origin(cfg.JournalBrowserURL); journal != "" {
		connect = append(connect, journal)
	}
	sources := map[string][]string{"connect-src": connect}
	if spa.Available(cfg.ClientDir) {
		hashes, err := spa.InlineScriptHashes(cfg.ClientDir)
		if err != nil {
			log.Warn("inline script hashes unavailable, the CSP will report the dashboard's scripts", slog.Any("error", err))
		}
		sources["script-src"] = hashes
	}
	return middleware.SecurityConfig{CSPSources: sources}
}

// origin returns the scheme and host of a URL, or "" when it is not an https one.
func origin(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
		return ""
	}
	return "https://" + parsed.Host
}
