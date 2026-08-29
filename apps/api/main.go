package main

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/FacileStudio/Journal/sdk/journal"
	"github.com/FacileStudio/porte/local"
	"github.com/FacileStudio/porte/oidc"
	portepg "github.com/FacileStudio/porte/pg"
	"github.com/FacileStudio/porte/session"
	"github.com/FacileStudio/tronc/apiref"
	apierrors "github.com/FacileStudio/tronc/errors"
	"github.com/FacileStudio/tronc/health"
	"github.com/FacileStudio/tronc/healthcheck"
	"github.com/FacileStudio/tronc/httpjson"
	"github.com/FacileStudio/tronc/httpx"
	"github.com/FacileStudio/tronc/logger"
	"github.com/FacileStudio/tronc/middleware"
	"github.com/FacileStudio/tronc/migrate"
	"github.com/FacileStudio/tronc/spa"
	"github.com/go-chi/chi/v5"

	"github.com/FacileStudio/Antenne/apps/api/internal/app"
	"github.com/FacileStudio/Antenne/apps/api/internal/database"
	"github.com/FacileStudio/Antenne/apps/api/internal/env"
	"github.com/FacileStudio/Antenne/apps/api/internal/server"
	"github.com/FacileStudio/Antenne/apps/api/modules/auth"
	"github.com/FacileStudio/Antenne/apps/api/modules/bus"
	"github.com/FacileStudio/Antenne/apps/api/modules/delivery"
	"github.com/FacileStudio/Antenne/apps/api/modules/events"
	// Aliased because tronc/health above answers "is this process alive", and this
	// one answers "can our delivery targets still be reached" — different
	// questions, and the import list is where that gets confused.
	targethealth "github.com/FacileStudio/Antenne/apps/api/modules/health"
	"github.com/FacileStudio/Antenne/apps/api/modules/settings"
	"github.com/FacileStudio/Antenne/apps/api/modules/spaces"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// refConfig describes the reference served at /docs.
//
// Servers is "/" rather than "/api" because Antenne's surface genuinely is not all
// under one prefix: the webhook receivers and the pool endpoints live at the
// root, where the systems that call them have them registered. Every route is
// therefore documented by its full path.
func refConfig() apiref.Config {
	return apiref.Config{
		Title:       "Antenne API",
		Description: "Alert aggregation and the Facile Suite's event bus. Configuration, the activity log, the third-party webhook intake and the Antenne Pool.",
		Servers:     []string{"/"},
		Registry: apiref.Registry{Modules: []apiref.Module{
			auth.Documentation(),
			server.SettingsDocumentation(),
			server.EventsDocumentation(),
			server.WebhooksDocumentation(),
			bus.Documentation(),
			spaces.Documentation(),
		}},
	}
}

func main() {
	// healthcheck runs before anything reads config, so a missing DATABASE_URL
	// cannot break the container probe.
	if healthcheck.Handle(os.Args) {
		return
	}

	os.Exit(run())
}

// run owns every defer and returns the process exit code. A bare return from
// main exits 0, which Docker reads as a clean stop and which hides the real
// cause of a startup failure.
func run() int {
	bootLogger := logger.New(logger.Config{})

	cfg, err := env.Load()
	if err != nil {
		bootLogger.Error("config", slog.Any("error", err))
		return 1
	}

	appLogger := logger.New(logger.Config{
		Level: cfg.LogLevel,
		Wrap:  journalWrap(cfg),
	})

	db, err := database.Open(cfg.DatabaseURL, appLogger)
	if err != nil {
		appLogger.Error("database", slog.Any("error", err))
		return 1
	}
	sqlDB, err := db.DB()
	if err != nil {
		appLogger.Error("database handle", slog.Any("error", err))
		return 1
	}
	defer sqlDB.Close()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	migrations, err := fs.Sub(migrationsFS, "migrations")
	if err != nil {
		appLogger.Error("migrations", slog.Any("error", err))
		return 1
	}
	migrateConfig := migrate.Config{DB: sqlDB, FS: migrations, Logger: appLogger}

	if handled, err := migrate.Command(ctx, os.Args, migrateConfig); handled {
		if err != nil {
			appLogger.Error("migrate", slog.Any("error", err))
			return 1
		}
		return 0
	}
	if err := migrate.Run(ctx, migrateConfig); err != nil {
		appLogger.Error("migrate", slog.Any("error", err))
		return 1
	}

	// One manager over one session table. Both kits issue through it, so a
	// password login and a federated one are the same row, the same cookie and
	// the same logout.
	identityStore := portepg.New(sqlDB)
	userStore := auth.NewStore(db)

	sessions, err := session.New(cfg.Porte, session.Deps{
		Sessions: identityStore.Sessions(),
		Logger:   appLogger,
	})
	if err != nil {
		appLogger.Error("sessions", slog.Any("error", err))
		return 1
	}

	authService := auth.NewService(db, appLogger, auth.Config{
		AdminUsername:     cfg.AdminUsername,
		AdminPassword:     cfg.AdminPassword,
		JournalBrowserURL: cfg.JournalBrowserURL,
		JournalBrowserKey: cfg.JournalBrowserKey,
	}, userStore, identityStore.Identities(), sessions)

	// oidc.New is the boot path: it performs discovery, so an unreachable or
	// half-configured issuer fails here rather than on somebody's first login
	// attempt. A kit with no OIDC_ISSUER is not an error — it still serves
	// /auth/config and authenticates every session this app issues.
	oidcKit, err := oidc.New(ctx, cfg.Porte, oidc.Deps{
		Users:       userStore,
		Identities:  identityStore.Identities(),
		Sessions:    sessions,
		Codes:       identityStore.LoginCodes(),
		Logger:      appLogger,
		ConfigExtra: authService.ConfigExtra(),
	})
	if err != nil {
		appLogger.Error("oidc", slog.Any("error", err))
		return 1
	}

	// Registration stays closed. The only local account is the bootstrap one
	// and it is seeded from the environment, so an open sign-up form on the
	// suite's event bus would be a way in that nobody asked for.
	passwords, err := local.New(local.Config{AllowRegistration: false}, local.Deps{
		Users:      userStore,
		Identities: identityStore.Identities(),
		Sessions:   sessions,
		Logger:     appLogger,
		Count:      userStore.CountUsers,
	})
	if err != nil {
		appLogger.Error("password login", slog.Any("error", err))
		return 1
	}

	if err := authService.EnsureAdmin(ctx); err != nil {
		appLogger.Error("auth", slog.Any("error", err))
		return 1
	}
	if oidcKit.Enabled() {
		appLogger.Info("single sign-on enabled",
			slog.String("issuer", cfg.Porte.Issuer),
			slog.Bool("sso_only", cfg.Porte.SSOOnly))
	}

	store, err := settings.NewStore(db, appLogger, settings.Defaults(cfg.Port, cfg.Timeout))
	if err != nil {
		appLogger.Error("settings", slog.Any("error", err))
		return 1
	}

	eventLog := events.NewService(db, appLogger)
	sender := delivery.NewSender(appLogger)
	spaceService := spaces.NewService(db)

	current := store.Get()
	application := app.New(appLogger, store, eventLog, sender, settings.RuntimeStatus{
		ServerPort: cfg.Port,
		// The dashboard still labels this "settings file"; the settings now
		// live in Postgres, so it reports the store rather than a path.
		SettingsPath:           "postgres",
		UsingPersistedSettings: store.Persisted(),
		DashboardBuilt:         spa.Available(cfg.ClientDir),
		LastReloadedAt:         time.Now().UTC().Format(time.RFC3339),
	})

	/*
	 * Target health has to be attached before Start, and without this call the
	 * whole subsystem is inert: a.health stays nil, every path is a nil-safe
	 * no-op, and nothing is ever probed. It is wired here rather than inside
	 * app.New because the store, the prober and the database handle are all
	 * assembled at this level.
	 *
	 * Probe is the sender's own method, so the thing that reports a target
	 * healthy is the thing that talks to it — a probe path that could drift from
	 * the send path would eventually certify a target the sender cannot reach.
	 */
	application.AttachHealth(app.HealthDeps{
		Records:           targethealth.NewStore(db),
		Probe:             sender.Probe,
		Policy:            targethealth.Evaluate,
		Ping:              sqlDB.PingContext,
		HeartbeatURL:      cfg.HeartbeatURL,
		HeartbeatInterval: cfg.HeartbeatInterval,
	})

	busServer := bus.NewServer(db, appLogger, busSecret(cfg, current))
	busServer.OnActivity(func(activity bus.Activity) {
		application.LogBusActivity(activity.Type, activity.App, activity.Detail)
	})
	busServer.OnEvent(func(appName, channel string, rawPayload []byte) {
		go func() {
			if strings.EqualFold(appName, "sonde") || strings.HasPrefix(channel, "monitor.") {
				application.DispatchBusEvent(context.Background(), settings.ProviderSonde, rawPayload)
			}
		}()
	})
	// A secret rotated from the dashboard takes effect on save, not on the next
	// restart — otherwise an operator changes it, sees it saved, and every pool
	// client keeps authenticating against the old one.
	application.OnSettingsApplied(func(next settings.AppSettings) {
		busServer.SetSecret(busSecret(cfg, next))
	})

	router := buildRouter(cfg, appLogger, sqlDB, application, authService, sessions, oidcKit, passwords, spaceService, busServer)
	// /dashboard is a legacy redirect, not API surface.
	if missing := apiref.Undocumented(router, refConfig(), "/dashboard"); len(missing) > 0 {
		appLogger.Warn("routes are missing from the API reference", slog.Any("routes", missing))
	}

	logStartup(appLogger, cfg, current, store)

	application.Start(ctx)
	defer application.Stop()

	go maintenance(ctx, appLogger, authService, eventLog, busServer)

	application.AnnounceStartup(ctx)

	httpServer := &http.Server{
		Addr:              ":" + strconv.Itoa(cfg.Port),
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
	}

	serverErrors := make(chan error, 1)
	go func() {
		appLogger.Info("listening", slog.Int("port", cfg.Port))
		serverErrors <- httpServer.ListenAndServe()
	}()

	select {
	case err := <-serverErrors:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			appLogger.Error("server", slog.Any("error", err))
			return 1
		}
	case <-ctx.Done():
		appLogger.Info("shutting down")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		appLogger.Error("shutdown", slog.Any("error", err))
		return 1
	}

	return 0
}

func buildRouter(
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
	apiref.Mount(router, refConfig())

	api := server.NewAPI(application)

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
	webhookHandler := server.NewWebhookHandler(application, log)
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

// maintenance runs the periodic chores: pool retention, session expiry and
// event-log pruning.
func maintenance(ctx context.Context, log *slog.Logger, authService *auth.Service, eventLog *events.Service, busServer *bus.Server) {
	daily := time.NewTicker(24 * time.Hour)
	hourly := time.NewTicker(time.Hour)
	defer daily.Stop()
	defer hourly.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-hourly.C:
			authService.PurgeExpiredSessions(ctx)
			eventLog.Prune()
		case <-daily.C:
			log.Debug("running pool retention")
			busServer.Cleanup()
		}
	}
}

func logStartup(log *slog.Logger, cfg env.Config, current settings.AppSettings, store *settings.Store) {
	counts := map[settings.ProviderType]int{}
	webhookCount := 0
	for _, provider := range current.Providers.Items {
		counts[provider.Type]++
		if provider.Type.IsWebhook() {
			webhookCount++
		}
	}

	log.Info("antenne booting",
		slog.String("app_env", string(cfg.AppEnv)),
		slog.Int("port", cfg.Port),
		slog.Bool("persisted_settings", store.Persisted()),
		slog.Bool("client_built", spa.Available(cfg.ClientDir)),
		slog.Int("delivery_targets", len(current.Delivery.Items)),
		slog.Int("providers", len(current.Providers.Items)),
		slog.Int("websites", counts[settings.ProviderWebsite]),
		slog.Int("imap", counts[settings.ProviderIMAP]),
		slog.Int("rss", counts[settings.ProviderRSS]),
		slog.Int("webhooks", webhookCount))

	if cfg.AdminPassword == "" {
		log.Warn("ANTENNE_ADMIN_PASSWORD is not set — the dashboard and its API are open to anyone who can reach this server")
	}
	log.Warn("rate limiting trusts X-Forwarded-For — make sure the reverse proxy overwrites it")
}

// poolSecret resolves the secret the pool accepts registrations under. The
// dashboard value wins over the environment, which is what lets an operator
// rotate it without a redeploy.
func busSecret(cfg env.Config, current settings.AppSettings) string {
	if current.Bus != nil && current.Bus.Secret != "" {
		return current.Bus.Secret
	}
	return cfg.PoolSecret
}

// journalWrap tees the application log to Journal when both variables are set.
// These few lines live here rather than in tronc on purpose: Journal is itself
// a tronc consumer, and making the chassis depend on its SDK would knot the two
// release cycles together.
func journalWrap(cfg env.Config) func(slog.Handler) slog.Handler {
	if cfg.JournalURL == "" || cfg.JournalToken == "" {
		return nil
	}
	client := journal.New(journal.Config{
		URL:   cfg.JournalURL,
		Token: cfg.JournalToken,
		App:   "antenne",
	})
	return func(next slog.Handler) slog.Handler {
		return journal.NewHandler(client, next)
	}
}
