package app_test

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/FacileStudio/Antenne/apps/api/internal/app"
	"github.com/FacileStudio/Antenne/apps/api/internal/pgtest"
	"github.com/FacileStudio/Antenne/apps/api/modules/delivery"
	"github.com/FacileStudio/Antenne/apps/api/modules/events"
	"github.com/FacileStudio/Antenne/apps/api/modules/settings"
	"gorm.io/gorm"
)

type replica struct {
	hits   atomic.Int32
	cancel context.CancelFunc
}

func startReplica(t *testing.T, db *gorm.DB) *replica {
	t.Helper()
	r := &replica{}
	site := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { r.hits.Add(1) }))
	t.Cleanup(site.Close)

	log := slog.New(slog.DiscardHandler)
	store, err := settings.NewStore(db, log, settings.Defaults(9090, 2))
	if err != nil {
		t.Fatal(err)
	}
	config := store.Get()
	provider := settings.DefaultProvider(settings.ProviderWebsite)
	provider.ID, provider.Name, provider.Enabled = "site", "Site", true
	provider.URL, provider.Interval = site.URL, 1
	config.Providers.Items = []settings.Provider{provider}
	raw, err := json.Marshal(config)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.Update(raw); err != nil {
		t.Fatal(err)
	}

	application := app.New(log, store, events.NewService(db, log), delivery.NewSender(log), settings.RuntimeStatus{})
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	r.cancel = cancel
	t.Cleanup(cancel)
	application.Start(ctx, sqlDB)
	return r
}

func eventually(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

func TestOnlyTheLockHolderRunsThePollers(t *testing.T) {
	db := pgtest.New(t)
	a := startReplica(t, db)
	b := startReplica(t, db)

	eventually(t, "a replica to poll", func() bool { return a.hits.Load()+b.hits.Load() >= 2 })
	time.Sleep(1500 * time.Millisecond)
	holder, waiter := a, b
	if a.hits.Load() == 0 {
		holder, waiter = b, a
	}
	if got := waiter.hits.Load(); got != 0 {
		t.Fatalf("the replica without the lock polled %d times", got)
	}

	holder.cancel()
	eventually(t, "the other replica to take over", func() bool { return waiter.hits.Load() >= 2 })
}
