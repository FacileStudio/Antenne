// Package server wires the HTTP surface: the dashboard API, the webhook
// intake, the pool endpoints and the SPA.
package server

import (
	"database/sql"
	"log/slog"
	"net/http"
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
	"github.com/FacileStudio/porte/local"
	"github.com/FacileStudio/porte/oidc"
	"github.com/FacileStudio/porte/session"
)

// BuildRouter is the composition root: wiring the health endpoint, the API
// reference, the three groups (public, auth-protected, sessioned) and the SPA
// catch-all. The caller passes the assembled dependencies and the router is the
// result.
func BuildRouter(
	cfg env.Config,
	log *slog.Logger,
	sqlDB *sql.DB,
	application *app.App,
	authService *auth.Service,
	sessions *session.Manager,
	oidcKit *oidc.Kit,
	passwords *local.Kit,
	spaceService *spaces.Service,
	busServer *bus.Server,
) chi.Router {
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
		CORS: middleware.CORSConfig{
			AllowedOrigins:   cfg.CORSAllowedOrigins,
			AllowCredentials: true,
		},
	})

	// Mounted at both the root and /api: an external monitor that only reaches
	// one of the two must not read the SPA shell as a green check.
	health.Mount(router, health.DB(sqlDB))

	// The reference stays at the root, unauthenticated, next to the routes it
	// documents. It describes the public contract — the webhook intake and the
	// pool protocol other apps integrate against — and carries no data.
	apiref.Mount(router, RefConfig())

	api := NewAPI(application)

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
		sessions.Mount(protected)
		oidcKit.Mount(protected)

		// Under SSO_ONLY the password route is not registered rather than
		// rejected, so there is no endpoint left to probe for an account.
		if authService.PasswordRequired() && !cfg.Porte.SSOOnly {
			passwords.Mount(protected.With(auth.ThrottleLogins()))
		}

		busServer.RegisterRoutes(protected)

		protected.Group(func(sessioned chi.Router) {
			sessioned.Use(auth.RequireAuth(sessions, authService))
			authService.RegisterRoutes(sessioned)
			api.RegisterRoutes(sessioned)
			spaceService.RegisterRoutes(sessioned)
			busServer.RegisterStats(sessioned)
		})
	})

	// The webhook paths are registered in third-party systems — Dokploy,
	// DocuSeal, a git forge — and cannot be moved under /api without breaking
	// every one of them. Same for /pool/ws, which pool clients pin.
	webhookHandler := NewWebhookHandler(application, log)
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

	return router
}
