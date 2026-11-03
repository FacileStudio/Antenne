package main

import (
	"context"
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
	"github.com/FacileStudio/porte/directory"
	"github.com/FacileStudio/porte/local"
	"github.com/FacileStudio/porte/oidc"
	portepg "github.com/FacileStudio/porte/pg"
	"github.com/FacileStudio/porte/session"
	"github.com/FacileStudio/tronc/apiref"
	"github.com/FacileStudio/tronc/healthcheck"
	"github.com/FacileStudio/tronc/logger"
	"github.com/FacileStudio/tronc/migrate"
	"github.com/FacileStudio/tronc/spa"

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
		if filled, err := userStore.BackfillPictures(ctx, cfg.Porte.Issuer); err != nil {
			appLogger.Warn("pictures not backfilled", slog.Any("error", err))
		} else {
			appLogger.Info("pictures backfilled", slog.Int64("users", filled))
		}
	}

	store, err := settings.NewStore(db, appLogger, settings.Defaults(cfg.Port, cfg.Timeout))
	if err != nil {
		appLogger.Error("settings", slog.Any("error", err))
		return 1
	}

	eventLog := events.NewService(db, appLogger)
	sender := delivery.NewSender(appLogger)
	spaceService := spaces.NewService(db, directory.Client{
		Issuer:       cfg.Porte.Issuer,
		ClientID:     cfg.Porte.ClientID,
		ClientSecret: cfg.Porte.ClientSecret,
	}, appLogger)

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

	router := server.BuildRouter(cfg, appLogger, sqlDB, application, authService, sessions, oidcKit, passwords, spaceService, busServer)
	// /dashboard is a legacy redirect, not API surface.
	if missing := apiref.Undocumented(router, server.RefConfig(), "/dashboard"); len(missing) > 0 {
		appLogger.Warn("routes are missing from the API reference", slog.Any("routes", missing))
	}

	logStartup(appLogger, cfg, current, store)

	application.Start(ctx)
	defer application.Stop()

	go maintenance(ctx, appLogger, authService, eventLog, busServer)
	go spaceService.Run(ctx)

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
